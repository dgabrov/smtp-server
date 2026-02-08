package srv

import (
	"context"
	"database/sql"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"
)

func (s *server) CopyMessages(ctx context.Context, set imap.NumSet, sourceMailboxID string, destinationMailboxID string) (*imap.CopyData, error) {
	// adjust numset
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var res imap.CopyData

	positionals, err := s.GetFilteredPositionalData(ctx, sourceMailboxID, set)
	if err != nil {
		return nil, err
	}

	var starterUID []imap.UID
	var endingUID []imap.UID

	for _, positional := range positionals {
		uid := positional.UID
		starterUID = append(starterUID, uid)

		// determine new uid
		newUID, err := getNextUid(ctx, tx, destinationMailboxID)
		if err != nil {
			return nil, err
		}

		err = copyMail(ctx, tx, positional.MessageID, newUID, destinationMailboxID)
		if err != nil {
			return nil, err
		}

		endingUID = append(endingUID, imap.UID(newUID))
	}

	res.UIDValidity = data.UIDValidity
	res.SourceUIDs = imap.UIDSetNum(starterUID...)
	res.DestUIDs = imap.UIDSetNum(endingUID...)

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func copyMail(ctx context.Context, tx *sql.Tx, originalMessageID string, newUID uint32, destinationMailboxID string) error {
	newMessageID := uuid.NewString()

	qr := `insert into message 
				(message_id, mailbox_id, body, uid, created_date, flag_seen, flag_answered, 
				 flag_flagged, flag_deleted, flag_draft) 
			select ?, ?, body, ?, created_date, flag_seen, 
				   flag_answered, flag_flagged, flag_deleted, flag_draft 
			from 
				message 
			where message_id = ?`
	_, err := tx.ExecContext(ctx, qr, newMessageID, destinationMailboxID, newUID, originalMessageID)

	return err
}
