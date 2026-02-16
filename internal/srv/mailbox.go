package srv

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
)

func (s *server) GetMailboxByName(ctx context.Context, userId string, name string) (*data.DmMailbox, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var res *data.DmMailbox

	qr := `select 
			   mailbox_id,
			   user_id,
			   name,
			   flag_non_existent,
			   flag_no_inferiors,
			   flag_no_select,
			   flag_marked,
			   flag_archive,
			   flag_drafts,
			   flag_flagged,
			   flag_junk,
			   flag_sent,
			   flag_trash,
			   flag_important
		from mailbox where user_id = ? and name = ?`

	st, err := tx.PrepareContext(ctx, qr)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	rs, err := st.QueryContext(ctx, userId, name)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	if rs.Next() {
		res, err = loadMailbox(rs)
	}

	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func loadMailbox(rs *sql.Rows) (*data.DmMailbox, error) {
	var mailboxID string
	var userID string
	var name string
	var flagNonExistent string
	var flagNoInferiors string
	var flagNoSelect string
	var flagMarked string
	var flagArchive, flagDraft string
	var flagFlagged, flagJunk, flagSent, flagTrash, flagImportant string

	err := rs.Scan(&mailboxID, &userID, &name, &flagNonExistent, &flagNoInferiors, &flagNoSelect, &flagMarked, &flagArchive, &flagDraft, &flagFlagged, &flagJunk, &flagSent, &flagTrash, &flagImportant)
	if err != nil {
		return nil, err
	}

	var flags []imap.MailboxAttr
	flags = processMailboxFlag(flags, flagNonExistent, imap.MailboxAttrNonExistent)
	flags = processMailboxFlag(flags, flagNoInferiors, imap.MailboxAttrNoInferiors)
	flags = processMailboxFlag(flags, flagNoSelect, imap.MailboxAttrNoSelect)
	flags = processMailboxFlag(flags, flagMarked, imap.MailboxAttrMarked)
	flags = processMailboxFlag(flags, flagArchive, imap.MailboxAttrArchive)
	flags = processMailboxFlag(flags, flagDraft, imap.MailboxAttrDrafts)
	flags = processMailboxFlag(flags, flagFlagged, imap.MailboxAttrFlagged)
	flags = processMailboxFlag(flags, flagJunk, imap.MailboxAttrJunk)
	flags = processMailboxFlag(flags, flagSent, imap.MailboxAttrSent)
	flags = processMailboxFlag(flags, flagTrash, imap.MailboxAttrTrash)
	flags = processMailboxFlag(flags, flagImportant, imap.MailboxAttrImportant)

	return &data.DmMailbox{
		MailboxID:  mailboxID,
		UserID:     userID,
		Name:       name,
		Attributes: flags,
	}, nil
}

func processMailboxFlag(flags []imap.MailboxAttr, strVal string, attribute imap.MailboxAttr) []imap.MailboxAttr {
	if strings.ToUpper(strVal) == "Y" {
		return append(flags, attribute)
	}

	return flags
}

func (s *server) GetMailboxStatus(ctx context.Context, mailboxID string) (*data.DmMailboxStatus, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	numMessages, err := getCountMessages(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	numRecent, err := getNumUnseen(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	uidNext, err := getNextUid(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	firstUnseenSeqNum, err := getFirstUnseenSeqNum(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	numDeleted, err := getNumDeleted(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &data.DmMailboxStatus{
		NumMessages:       numMessages,
		FirstUnseenSeqNum: firstUnseenSeqNum,
		NumRecent:         numRecent,
		UIDNext:           uidNext,
		UIDValidity:       data.UIDValidity,
		NumDeleted:        numDeleted,
		NumUnseen:         numRecent,
	}, nil
}

func getNumUnseen(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, "select count(*) from message where mailbox_id = ? and flag_seen = 'N'", mailboxID)
	if err != nil {
		return 0, err
	}
	defer rs.Close()

	var val uint32
	rs.Next()
	err = rs.Scan(&val)
	if err != nil {
		return 0, err
	}

	return val, nil
}

func getNumDeleted(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, "select count(*) from message where mailbox_id = ? and flag_deleted = 'Y'", mailboxID)
	if err != nil {
		return 0, err
	}

	var val uint32
	rs.Next()
	err = rs.Scan(&val)
	if err != nil {
		return 0, err
	}

	return val, nil
}

func getCountMessages(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, "SELECT count(*) FROM message WHERE mailbox_id = ?", mailboxID)
	if err != nil {
		return 0, err
	}
	defer rs.Close()

	var val uint32
	rs.Next()
	err = rs.Scan(&val)
	if err != nil {
		return 0, err
	}

	return val, nil
}

func getFirstUnseenSeqNum(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, `select row_num
			from (select row_number() over (order by created_date, uid) row_num, flag_seen from message where mailbox_id = ?) m
			where m.flag_seen = 'N' order by row_num
			limit 1`, mailboxID)
	if err != nil {
		return 0, err
	}
	defer rs.Close()

	var val uint32 = 0
	if rs.Next() {
		err = rs.Scan(&val)
		if err != nil {
			return 0, err
		}
	}

	return val, nil
}

func getNextUid(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	qr := "select max(uid) from message where mailbox_id = ?"
	var res uint32
	res = 1

	st, err := tx.PrepareContext(ctx, qr)
	if err != nil {
		return 0, err
	}
	defer st.Close()

	rows, err := st.QueryContext(ctx, mailboxID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var pulledVal sql.NullInt32

	if rows.Next() {
		err := rows.Scan(&pulledVal)
		if err != nil {
			return 0, err
		}

		if pulledVal.Valid {
			res = uint32(pulledVal.Int32) + 1
		}
	}

	return res, nil
}

func (s *server) GetMessageCount(ctx context.Context, mailboxID string) (uint32, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	number, err := getCountMessages(ctx, tx, mailboxID)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return number, nil
}

func (s *server) GetChildMailboxes(ctx context.Context, userID string, mailboxID string) ([]*data.DmMailbox, error) {
	// first load the first one
	mbox, err := s.GetMailboxByID(ctx, userID, mailboxID)
	if err != nil {
		return nil, err
	}

	if mbox == nil {
		return nil, fmt.Errorf("mailbox id: %s not found", mailboxID)
	}

	// ok, get the name and proceed
	startWith := mbox.Name + data.MailboxSeparator

	return s.GetMailboxesNameStartWith(ctx, userID, mailboxID, startWith)
}

func (s *server) GetMailboxesNameStartWith(ctx context.Context, userID string, mailboxID string, startWith string) ([]*data.DmMailbox, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	qr := `select mailbox_id,
				   mailbox_id,
				   user_id,
				   name,
				   flag_non_existent,
				   flag_no_inferiors,
				   flag_no_select,
				   flag_marked,
				   flag_archive,
				   flag_drafts,
				   flag_flagged,
				   flag_junk,
				   flag_sent,
				   flag_trash,
				   flag_important
			from mailbox
			where user_id = ?
			  and mailbox_id != ?
			  and name like ?`

	rs, err := tx.QueryContext(ctx, qr, userID, mailboxID, startWith+"%")
	if err != nil {
		return nil, err
	}

	defer rs.Close()

	var res []*data.DmMailbox

	for rs.Next() {
		mbox, err := loadMailbox(rs)
		if err != nil {
			return nil, err
		}

		res = append(res, mbox)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *server) CreateMailbox(ctx context.Context, userID string, newMailboxID string, mailbox string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	qr := "insert into mailbox (mailbox_id, user_id, name) values (?, ?, ?)"
	_, err = tx.ExecContext(ctx, qr, newMailboxID, userID, mailbox)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *server) UpdateMailboxName(ctx context.Context, userID string, mailboxID string, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	_, err = tx.ExecContext(ctx, "update mailbox set name = ? where mailbox_id = ? and user_id = ?", name, mailboxID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *server) ListMailboxes(ctx context.Context, userID string) ([]*data.DmMailbox, error) {
	qr := `select 
				   mailbox_id,
				   user_id,
				   name,
				   flag_non_existent,
				   flag_no_inferiors,
				   flag_no_select,
				   flag_marked,
				   flag_archive,
				   flag_drafts,
				   flag_flagged,
				   flag_junk,
				   flag_sent,
				   flag_trash,
				   flag_important
			from mailbox
			where user_id = ? order by name`

	rs, err := s.db.QueryContext(ctx, qr, userID)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var res []*data.DmMailbox
	for rs.Next() {
		mbox, err := loadMailbox(rs)
		if err != nil {
			return nil, err
		}

		res = append(res, mbox)
	}

	return res, nil
}

func (s *server) DeleteMailbox(ctx context.Context, userID string, mailboxID string) error {
	_, err := s.db.ExecContext(ctx, "delete from mailbox where mailbox_id = ? and user_id = ?", mailboxID, userID)

	return err
}
