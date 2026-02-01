package srv

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/google/uuid"
)

type Servr interface {
	Authenticate(context context.Context, username string, password string) error
	GetLoginAndDomain(context context.Context, email string) (string, string, error)
	IsLocalDomain(context context.Context, domain string) (bool, error)
	CheckLocalUserAndDomain(context context.Context, login string, domain string) error
	DeliverLocally(context context.Context, to string, body []byte) error
	DeliverQueue(context context.Context, from string, to []string, bytes []byte) error
	MarkQueueItemSuccess(context context.Context, queueItemID string) error
	AddQueueItemError(ctx context.Context, queueItemID string) error
	LoadQueueItemByID(ctx context.Context, id string) (*data.DmQueue, error)
	LoadQueueRecipients(ctx context.Context, config data.QueueConfig) ([]*data.DmQueueRecipient, error)
	GetUserID(ctx context.Context, email string) (string, error)
	GetMailboxByName(ctx context.Context, userID string, name string) (*data.DmMailbox, error)
	GetMailboxStatus(ctx context.Context, mailboxID string) (*data.DmMailboxStatus, error)
	GetMessageCount(ctx context.Context, mailboxID string) (uint32, error)
	GetChildMailboxes(ctx context.Context, userID string, mailboxID string) ([]*data.DmMailbox, error)
}

func NewServer(db *sql.DB) Servr {
	return &server{db: db}
}

type server struct {
	db *sql.DB
}

func (s *server) GetUserID(ctx context.Context, email string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	login, domain, err := getLoginAndDomain(email)
	if err != nil {
		return "", err
	}

	userID, err := getUserID(ctx, tx, login, domain)
	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (s *server) LoadQueueRecipients(ctx context.Context, config data.QueueConfig) ([]*data.DmQueueRecipient, error) {
	// condition to load the values are 1. success_ind not true, 2. last attempted smaller than the current timestamp minus delay
	// no more than whatever slice size

	qr := `select queue_recipient_id, queue_id, to_addr, attempts, 
       last_attempted_dt, success_ind 
			from queue_recipient 
			where success_ind != 'Y' 
			and (last_attempted_dt is null or last_attempted_dt < ?) and attempts < ? limit ?`

	timeProcessing := time.Now().Add(time.Duration(-config.TimeBetweenAttempts) * time.Second)

	rs, err := s.db.QueryContext(ctx, qr, timeProcessing, config.MaxAttempts, config.LoadSize)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var res []*data.DmQueueRecipient

	for rs.Next() {
		item := data.DmQueueRecipient{}

		var s string
		var lastAttemptedDt sql.NullTime

		err = rs.Scan(&item.QueueRecipientID, &item.QueueID, &item.ToAddr, &item.Attempts, &lastAttemptedDt, &s)

		item.Success = strings.ToUpper(s) == "Y"
		item.LastAttemptedDt = lastAttemptedDt.Time

		res = append(res, &item)
	}

	return res, nil

}

func (s *server) LoadQueueItemByID(ctx context.Context, id string) (*data.DmQueue, error) {
	var res data.DmQueue
	qr := "select queue_id, from_addr, body from queue where queue_id = ?"
	st, err := s.db.PrepareContext(ctx, qr)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	rows, err := st.QueryContext(ctx, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		res = data.DmQueue{}

		err = rows.Scan(&res.QueueID, &res.From, &res.Body)
		if err != nil {
			return nil, err
		}

		return &res, nil
	}

	return nil, fmt.Errorf("queue item with the id %s does not exist", id)
}

func (s *server) MarkQueueItemSuccess(ctx context.Context, queueItemID string) error {
	now := time.Now()
	qr := "update queue_recipient set success_ind = 'Y', last_attempted_dt = ? where queue_recipient_id = ?"
	st, err := s.db.PrepareContext(ctx, qr)
	if err != nil {
		return err
	}
	defer st.Close()

	_, err = st.ExecContext(ctx, now, queueItemID)

	return err
}

func (s *server) AddQueueItemError(ctx context.Context, queueItemID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	nrAttempts, err := getNrAttempts(ctx, tx, queueItemID)
	if err != nil {
		return err
	}

	err = updateQueueItem(ctx, tx, queueItemID, nrAttempts+1)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func updateQueueItem(ctx context.Context, tx *sql.Tx, queueItemID string, nrAttempts int) error {
	qr := "update queue_recipient set attempts = ?, last_attempted_dt = ? where queue_recipient_id = ?"

	_, err := tx.ExecContext(ctx, qr, nrAttempts, time.Now(), queueItemID)

	return err
}

func getNrAttempts(ctx context.Context, tx *sql.Tx, id string) (int, error) {
	qr := "select attempts from queue_recipient where queue_recipient_id = ?"
	var nr int
	st, err := tx.PrepareContext(ctx, qr)
	if err != nil {
		return 0, err
	}
	defer st.Close()
	rows, err := st.QueryContext(ctx, id)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&nr)

		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("queue item with the id %s does not exist", id)
	}

	return nr, nil
}

func (s *server) DeliverLocally(context context.Context, to string, body []byte) error {
	tx := getTx(context, s.db)
	defer tx.Rollback()

	login, domain, err := getLoginAndDomain(to)
	if err != nil {
		return err
	}

	userID, err := getUserID(context, tx, login, domain)
	if len(userID) == 0 {
		return errors.New("user not found")
	}

	mailboxID, err := getMailboxID(context, tx, userID)
	if err != nil {
		return err
	}

	if len(mailboxID) == 0 {
		return errors.New("mailbox not found")
	}

	uid, err := getNextUid(context, tx, mailboxID)
	if err != nil {
		return err
	}

	messageID := uuid.NewString()

	qr := "insert into message (message_id, mailbox_id, body, uid, created_date) values (?, ?, ?, ?, ?)"
	_, err = tx.ExecContext(context, qr, messageID, mailboxID, body, uid, time.Now().UTC())
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *server) DeliverQueue(context context.Context, from string, to []string, bytes []byte) error {
	if len(to) == 0 {
		// no recipients to be relayed
		return nil
	}

	tx := getTx(context, s.db)
	defer tx.Rollback()

	qr := "insert into queue (queue_id, from_addr, body) values (?, ?, ?)"
	id := uuid.NewString()

	_, err := tx.Exec(qr, id, from, bytes)
	if err != nil {
		return err
	}

	// and now the recipients
	qr = "insert into queue_recipient (queue_recipient_id, queue_id, to_addr) values (?, ?, ?)"
	for _, toAddress := range to {
		qrID := uuid.NewString()
		_, err = tx.Exec(qr, qrID, id, toAddress)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *server) CheckLocalUserAndDomain(context context.Context, login string, domain string) error {
	tx := getTx(context, s.db)
	defer tx.Rollback()

	userID, err := getUserID(context, tx, login, domain)
	if err != nil {
		return err
	}

	if len(userID) == 0 {
		return errors.New("user not found")
	}

	return tx.Commit()
}

func getUserID(ctx context.Context, tx *sql.Tx, login string, domain string) (string, error) {
	dmDomain, err := getDomainByName(ctx, tx, domain)
	if err != nil {
		return "", err
	}
	if dmDomain == nil {
		return "", errors.New("domain not found")
	}

	dmUser, err := getDestinationUser(ctx, tx, dmDomain.DomainID, login)
	if err != nil {
		return "", err
	}

	if dmUser == nil {
		// user not found, try alternate
		slog.InfoContext(ctx, fmt.Sprintf("user %s not found, attempt alternate", login))

		if dmDomain.CatchAll {
			alternateLogin := dmDomain.CatchAllLogin

			dmAlternateUser, err := getDestinationUser(ctx, tx, dmDomain.DomainID, alternateLogin)
			if err != nil {
				return "", err
			}

			if dmAlternateUser == nil {
				return "", fmt.Errorf("alternate user %s not found", alternateLogin)
			}

			slog.InfoContext(ctx, fmt.Sprintf("alternate user %s found", alternateLogin))

			return dmAlternateUser.UserID, nil
		} else {
			return "", fmt.Errorf("domain %s does not have catch all user", dmDomain.Name)
		}
	} else {
		slog.InfoContext(ctx, fmt.Sprintf("found user: %s", login))

		return dmUser.UserID, nil
	}

}

func getDomainByName(ctx context.Context, tx *sql.Tx, domain string) (*data.DmDomain, error) {
	dm := strings.ToLower(domain)
	qr := "select domain_id, name, catchall_ind, catchall_login from domain where name = ?"
	rs, err := tx.QueryContext(ctx, qr, dm)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	if rs.Next() {
		val := data.DmDomain{}
		isCatchAll := ""
		err = rs.Scan(&val.DomainID, &val.Name, &isCatchAll, &val.CatchAllLogin)
		if err != nil {
			return nil, err
		}

		val.CatchAll = isCatchAll == "Y"

		return &val, nil
	}

	return nil, nil
}

func getDestinationUser(ctx context.Context, tx *sql.Tx, domainID string, user string) (*data.DmUser, error) {
	usr := strings.ToLower(user)
	qr := "select user_id, domain_id, login, password from mailbox_user where domain_id = ? and login = ?"

	rs, err := tx.QueryContext(ctx, qr, domainID, usr)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	if rs.Next() {
		val := data.DmUser{}
		err = rs.Scan(&val.UserID, &val.DomainID, &val.Login, &val.Password)
		if err != nil {
			return nil, err
		}

		return &val, nil
	}

	return nil, nil
}

func (s *server) IsLocalDomain(context context.Context, domain string) (bool, error) {
	tx := getTx(context, s.db)
	defer tx.Rollback()

	dmDomain, err := getDomainByName(context, tx, domain)
	if err != nil {
		return false, err
	}

	res := dmDomain != nil

	err = tx.Commit()

	return res, err
}

func (s *server) GetLoginAndDomain(context context.Context, email string) (string, string, error) {
	return getLoginAndDomain(email)
}

func (s *server) Authenticate(context context.Context, username string, password string) error {
	hashed := hashPassword(password)
	tx := getTx(context, s.db)
	defer tx.Rollback()

	login, domain, err := getLoginAndDomain(username)
	if err != nil {
		return err
	}

	qr := "select mu.password from mailbox_user mu, domain d where mu.domain_id = d.domain_id and d.name = ? and mu.login = ?"
	var pass string

	rs, err := tx.QueryContext(context, qr, strings.ToLower(domain), strings.ToLower(login))
	if err != nil {
		return err
	}
	defer rs.Close()

	if rs.Next() {
		err = rs.Scan(&pass)
		if err != nil {
			return err
		}

		if pass != hashed {
			return errors.New("invalid credentials")
		}
	} else {
		return errors.New("user not found")
	}

	return tx.Commit()
}

func getLoginAndDomain(email string) (string, string, error) {
	em := strings.ToLower(email)
	items := strings.Split(em, "@")
	if len(items) != 2 {
		return "", "", errors.New("invalid email")
	}

	return items[0], items[1], nil
}

func getMailboxID(ctx context.Context, tx *sql.Tx, userID string) (string, error) {
	qr := "select mailbox_id from mailbox where name = 'INBOX' && user_id = ?"
	rs, err := tx.QueryContext(ctx, qr, userID)
	if err != nil {
		return "", err
	}

	defer rs.Close()

	mailboxID := ""

	if rs.Next() {
		err = rs.Scan(&mailboxID)
		if err != nil {
			return "", err
		}
	}

	return mailboxID, nil
}

func hashPassword(password string) string {
	hash := sha512.Sum512([]byte(password))

	return fmt.Sprintf("%x", hash) // %x automatically converts bytes to hex
}

func getTx(ctx context.Context, db *sql.DB) *sql.Tx {
	tx, err := db.Begin()
	if err != nil {
		slog.ErrorContext(ctx, "cannot secure transaction")
	}

	return tx
}
