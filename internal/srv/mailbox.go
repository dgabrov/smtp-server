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
	defer tx.Rollback()
	var res *data.DmMailbox

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
	var flagJunk, flagSent, flagTrash, flagImportant string

	err := rs.Scan(&mailboxID, &userID, &name, &flagNonExistent, &flagNoInferiors, &flagNoSelect, &flagMarked, &flagArchive, &flagDraft, &flagJunk, &flagSent, &flagTrash, &flagImportant)
	if err != nil {
		return nil, err
	}

	var flags []imap.MailboxAttr
	flags = processFlag(flags, flagNonExistent, imap.MailboxAttrNonExistent)
	flags = processFlag(flags, flagNoInferiors, imap.MailboxAttrNoInferiors)
	flags = processFlag(flags, flagNoSelect, imap.MailboxAttrNoSelect)
	flags = processFlag(flags, flagMarked, imap.MailboxAttrMarked)
	flags = processFlag(flags, flagArchive, imap.MailboxAttrArchive)
	flags = processFlag(flags, flagDraft, imap.MailboxAttrDrafts)
	flags = processFlag(flags, flagJunk, imap.MailboxAttrJunk)
	flags = processFlag(flags, flagSent, imap.MailboxAttrSent)
	flags = processFlag(flags, flagTrash, imap.MailboxAttrTrash)
	flags = processFlag(flags, flagImportant, imap.MailboxAttrImportant)

	return &data.DmMailbox{
		MailboxID:  mailboxID,
		UserID:     userID,
		Name:       name,
		Attributes: flags,
	}, nil
}

func processFlag(flags []imap.MailboxAttr, strVal string, attribute imap.MailboxAttr) []imap.MailboxAttr {
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
	defer tx.Rollback()

	numMessages, err := getNumMessages(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	numRecent, err := getNumRecent(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	uidNext, err := getNextUid(ctx, tx, mailboxID)
	if err != nil {
		return nil, err
	}

	firstUnseenSeqNum, err := getFirstUnseenSeqNum(ctx, tx, mailboxID)

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
	}, nil
}

func getNumMessages(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, "SELECT count(*) FROM message WHERE mailbox_id = ?", mailboxID)
	if err != nil {
		return 0, err
	}
	defer rs.Close()

	var val uint32
	err = rs.Scan(&val)
	if err != nil {
		return 0, err
	}

	return val, nil
}

func getFirstUnseenSeqNum(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, `select row_num
			from (select row_number() over (order by created_date) row_num, flag_seen from message where mailbox_id = ?) m
			where m.flag_seen = 'N'
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

func getNumRecent(ctx context.Context, tx *sql.Tx, mailboxID string) (uint32, error) {
	rs, err := tx.QueryContext(ctx, "SELECT count(*) FROM message WHERE mailbox_id = ? and flag_seen != 'Y'", mailboxID)
	if err != nil {
		return 0, err
	}
	defer rs.Close()

	var val uint32
	err = rs.Scan(&val)

	if err != nil {
		return 0, err
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
	defer tx.Rollback()
	number, err := getNumMessages(ctx, tx, mailboxID)
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
	mbox, err := s.GetMailboxByName(ctx, userID, mailboxID)
	if err != nil {
		return nil, err
	}

	if mbox == nil {
		return nil, fmt.Errorf("mailbox id: %s not found", mailboxID)
	}

	// ok, get the name and proceed
	startWith := mbox.Name + data.MailboxSeparator

	return getMailboxesNameStartWith(ctx, userID, mailboxID, startWith)
}

// the idea is that the other mailboxes must be different than the existent one
func getMailboxesNameStartWith(ctx context.Context, userID string, mailboxID string, startWith string) ([]*data.DmMailbox, error) {
	// TODO continue here
	return nil, nil
}
