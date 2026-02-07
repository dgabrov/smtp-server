package srv

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"
)

func (s *server) GetExpungeInformation(ctx context.Context, uids *imap.UIDSet, mailboxID string) ([]*data.SeqHolder, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qr := `select row_nr, message_id
		from (select row_number() over (order by created_date) row_nr, message_id, uid
			  from message
			  where flag_deleted = 'Y'
				and mailbox_id = ?) dlt `

	// very bad : load all of them and then filter
	rs, err := tx.QueryContext(ctx, qr, mailboxID)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var allItems []*data.SeqHolder

	for rs.Next() {
		var item data.SeqHolder
		err := rs.Scan(&item.NumSeq, &item.ID, &item.UID)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, &item)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	var results []*data.SeqHolder
	if uids != nil {
		err = dealWithWildcards(uids, nil, func() (imap.UID, error) {
			val, err := getNextUid(ctx, tx, mailboxID)
			if err != nil {
				return 0, err
			}

			return imap.UID(val - 1), nil
		})

		if err != nil {
			return nil, err
		}

		for _, seqHolder := range allItems {
			uid := seqHolder.UID

			if uids.Contains(imap.UID(uid)) {
				results = append(results, seqHolder)
			}
		}
	} else {
		results = allItems
	}

	return results, nil
}

func (s *server) DeleteMessage(ctx context.Context, mailboxID string, messageID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "delete from message where mailbox_id = ? and message_id = ?", mailboxID, messageID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *server) AppendMessage(ctx context.Context, mailboxID string, body []byte, flags []imap.Flag, t time.Time) (imap.UID, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	uid, err := getNextUid(ctx, tx, mailboxID)
	if err != nil {
		return 0, err
	}

	messageID := uuid.NewString()
	createdDate := t.UTC()
	flagSeen := 'N'
	flagAnswered := 'N'
	flagFlagged := 'N'
	flagDeleted := 'N'
	flagDraft := 'N'

	if slices.Contains(flags, imap.FlagSeen) {
		flagSeen = 'Y'
	}

	if slices.Contains(flags, imap.FlagAnswered) {
		flagAnswered = 'Y'
	}

	if slices.Contains(flags, imap.FlagFlagged) {
		flagFlagged = 'Y'
	}

	if slices.Contains(flags, imap.FlagDeleted) {
		flagDeleted = 'Y'
	}

	if slices.Contains(flags, imap.FlagDraft) {
		flagDraft = 'Y'
	}

	_, err = tx.ExecContext(ctx, `insert into message (message_id, mailbox_id, body, uid, 
                     created_date, flag_seen, flag_answered, flag_flagged,
                     flag_deleted, flag_draft)
				values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, messageID, mailboxID, body, uid,
		createdDate, flagSeen, flagAnswered, flagFlagged, flagDeleted, flagDraft)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return imap.UID(uid), nil
}

func (s *server) GetFilteredPositionalData(ctx context.Context, mailboxID string, set imap.NumSet) ([]*data.DmPositionalMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var res []*data.DmPositionalMessage

	qr := "select row_number() over (order by created_date) row_num, message_id, uid from message where mailbox_id = ?"
	rs, err := tx.QueryContext(ctx, qr, mailboxID)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	for rs.Next() {
		var item data.DmPositionalMessage
		err = rs.Scan(&item.SeqNum, &item.MessageID, &item.UID)

		if err != nil {
			return nil, err
		}

		res = append(res, &item)
	}

	// ok now filter the values
	err = dealWithWildcards(set, func() (uint32, error) {
		return getCountMessages(ctx, tx, mailboxID)
	}, func() (imap.UID, error) {
		uid, err := getNextUid(ctx, tx, mailboxID)
		if err != nil {
			return 0, err
		}

		return imap.UID(uid - 1), nil
	})

	if err != nil {
		return nil, err
	}

	// and now filter
	ns, ok := set.(imap.SeqSet)
	if ok {
		var items []*data.DmPositionalMessage
		for _, item := range res {
			if ns.Contains(item.SeqNum) {
				items = append(items, item)
			}
		}

		return items, nil
	}

	uidSet, ok := set.(imap.UIDSet)
	if ok {
		var items []*data.DmPositionalMessage

		for _, item := range res {
			if uidSet.Contains(item.UID) {
				items = append(items, item)
			}
		}

		return items, nil
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	// just get all of them
	return res, nil
}

func (s *server) GetMessageBody(ctx context.Context, messageID string) ([]byte, error) {
	var blob []byte
	qr := "select body from message where message_id = ?"
	rs, err := s.db.QueryContext(ctx, qr, messageID)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	if rs.Next() {
		err = rs.Scan(&blob)
		if err != nil {
			return nil, err
		}
	} else {
		err = errors.New("message not found")

		return nil, err
	}

	return blob, nil
}

func (s *server) GetStrippedMessages(ctx context.Context, nset imap.NumSet, mailboxID string) ([]*data.DmStrippedMessage, error) {
	positional, err := s.GetFilteredPositionalData(ctx, mailboxID, nset)
	if err != nil {
		return nil, err
	}

	var ids []any
	positionalMap := make(map[string]*data.DmPositionalMessage)

	for _, m := range positional {
		ids = append(ids, m.MessageID)

		positionalMap[m.MessageID] = m
	}

	if len(positional) == 0 {
		return nil, nil
	}

	// and now we retrieve the messages just like
	qr := "select message_id, mailbox_id, created_date, flag_seen, flag_answered, flag_flagged, flag_deleted, flag_draft from message where message_id in (?" + strings.Repeat(",?", len(positional)-1) + ")"
	rs, err := s.db.QueryContext(ctx, qr, ids...)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var res []*data.DmStrippedMessage

	for rs.Next() {
		var messageID string
		var mboxID string
		var createdDate time.Time
		var flagSeen string
		var flagAnswered string
		var flagFlagged string
		var flagDeleted string
		var flagDraft string

		err = rs.Scan(&messageID, &mboxID, &createdDate, &flagSeen, &flagAnswered, &flagFlagged, &flagDeleted, &flagDraft)
		if err != nil {
			return nil, err
		}

		var flags []imap.Flag
		flags = processMessageFlag(flags, flagSeen, imap.FlagSeen)
		flags = processMessageFlag(flags, flagAnswered, imap.FlagAnswered)
		flags = processMessageFlag(flags, flagFlagged, imap.FlagFlagged)
		flags = processMessageFlag(flags, flagDeleted, imap.FlagDeleted)
		flags = processMessageFlag(flags, flagDraft, imap.FlagDraft)

		pos, ok := positionalMap[messageID]
		if !ok {
			return nil, errors.New("message not found in the map, probably deleted in the meantime")
		}

		item := &data.DmStrippedMessage{
			MessageID: messageID,
			MailboxID: mboxID,
			UID:       pos.UID,
			SeqNum:    pos.SeqNum,
			Flags:     flags,
		}

		res = append(res, item)
	}

	return res, nil
}

func (s *server) SearchMessages(ctx context.Context, mailboxID string, search string) ([]uint32, []imap.UID, error) {
	qr := "select row_number() over (order by created_date) numseq, uid from message where mailbox_id = ? and body like ?"

	sr := search
	if !strings.HasPrefix(search, "%") {
		sr = "%" + sr
	}

	if !strings.HasSuffix(search, "%") {
		sr = "%" + sr
	}

	rs, err := s.db.QueryContext(ctx, qr, mailboxID, sr)
	if err != nil {
		return nil, nil, err
	}
	defer rs.Close()

	var numSeq uint32
	var uid imap.UID

	var resSeq []uint32
	var uids []imap.UID

	for rs.Next() {
		err = rs.Scan(&numSeq, &uid)
		if err != nil {
			return nil, nil, err
		}
		resSeq = append(resSeq, numSeq)
		uids = append(uids, uid)
	}

	return resSeq, uids, nil
}
