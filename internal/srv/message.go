package srv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message/mail"
	"github.com/google/uuid"
)

func (s *server) GetExpungeInformation(ctx context.Context, uids *imap.UIDSet, mailboxID string) ([]*data.SeqHolder, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	qr := `select row_number() over (order by created_date, uid) row_nr, message_id, uid
			  from message
			  where flag_deleted = 'Y'
				and mailbox_id = ? order by row_nr`

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

	allItems = filterSeqHolders(allItems, uids)

	return allItems, nil
}

func (s *server) DeleteMessage(ctx context.Context, mailboxID string, messageID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

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
	defer rollback(tx)

	uid, err := getNextUid(ctx, tx, mailboxID)
	if err != nil {
		return 0, err
	}

	messageID := uuid.NewString()
	createdDate := t.UTC()
	flagSeen := "N"
	flagAnswered := "N"
	flagFlagged := "N"
	flagDeleted := "N"
	flagDraft := "N"

	if slices.Contains(flags, imap.FlagSeen) {
		flagSeen = "Y"
	}

	if slices.Contains(flags, imap.FlagAnswered) {
		flagAnswered = "Y"
	}

	if slices.Contains(flags, imap.FlagFlagged) {
		flagFlagged = "Y"
	}

	if slices.Contains(flags, imap.FlagDeleted) {
		flagDeleted = "Y"
	}

	if slices.Contains(flags, imap.FlagDraft) {
		flagDraft = "Y"
	}

	subject, from, sender, replyTo, to, cc, bcc, inReplyTo, messageDate, parseErr := parseEmailHeaders(body)
	if parseErr != nil {
		slog.WarnContext(ctx, fmt.Sprintf("failed to parse email headers: %v", parseErr))
	}

	_, err = tx.ExecContext(ctx, `insert into message (message_id, mailbox_id, body, uid,
                     created_date, flag_seen, flag_answered, flag_flagged,
                     flag_deleted, flag_draft, message_date, subject, message_from, sender, reply_to, message_to, cc, bcc, in_reply_to)
				values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, messageID, mailboxID, body, uid,
		createdDate, flagSeen, flagAnswered, flagFlagged, flagDeleted, flagDraft, messageDate, subject, from, sender, replyTo, to, cc, bcc, inReplyTo)
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
	defer rollback(tx)

	var res []*data.DmPositionalMessage

	qr := "select row_number() over (order by created_date, uid) row_num, message_id, uid from message where mailbox_id = ? order by row_num"
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

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	res = filterPositionalMessages(res, set)

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
	var strIds []string
	resultMap := make(map[string]*data.DmStrippedMessage)
	posMap := make(map[string]*data.DmPositionalMessage)

	for _, m := range positional {
		ids = append(ids, m.MessageID)
		strIds = append(strIds, m.MessageID)

		posMap[m.MessageID] = m
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

		item := &data.DmStrippedMessage{
			MessageID: messageID,
			MailboxID: mboxID,
			Flags:     flags,
		}

		resultMap[messageID] = item
	}

	// go through them
	for _, id := range strIds {
		item := resultMap[id]

		pos, ok := posMap[id]
		if ok {
			item.SeqNum = pos.SeqNum
			item.UID = pos.UID
		}

		if ok {
			res = append(res, item)
		}
	}

	return res, nil
}

func (s *server) SearchMessages(ctx context.Context, mailboxID string, criteria *imap.SearchCriteria) ([]uint32, []imap.UID, error) {
	query := "select row_number() over (order by created_date, uid) numseq, uid from message where mailbox_id = ?"
	args := []interface{}{mailboxID}

	if criteria != nil {
		whereClause, params := buildSearchWhere(criteria)
		if whereClause != "" {
			query += " and " + whereClause
			args = append(args, params...)
		}
	}

	query += " order by numseq"

	rs, err := s.db.QueryContext(ctx, query, args...)
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

func buildSearchWhere(criteria *imap.SearchCriteria) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if len(criteria.Body) > 0 {
		for _, term := range criteria.Body {
			conditions = append(conditions, "body like ?")
			args = append(args, "%"+term+"%")
		}
	}

	if len(criteria.Text) > 0 {
		for _, term := range criteria.Text {
			search := "%" + term + "%"
			conditions = append(conditions, "(body like ? or subject like ? or message_from like ? or message_to like ?)")
			args = append(args, search, search, search, search)
		}
	}

	var whereClause string
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " or ")
	}

	return whereClause, args
}

func (s *server) MarkMessageAsSeen(ctx context.Context, messageID string) error {
	slog.InfoContext(ctx, fmt.Sprintf("marking message as seen: %s", messageID))

	qr := `UPDATE message SET flag_seen = 'Y' WHERE message_id = ?`
	_, err := s.db.ExecContext(ctx, qr, messageID)

	return err
}

func addressesToString(addrs []*mail.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	var result []string
	for _, addr := range addrs {
		result = append(result, addr.Address)
	}
	return strings.Join(result, ",")
}

func parseEmailHeaders(body []byte) (subject, from, sender, replyTo, to, cc, bcc, inReplyTo *string, messageDate *time.Time, err error) {
	r, err := mail.CreateReader(bytes.NewReader(body))
	if err != nil {
		return
	}

	header := r.Header

	if subj, err := header.Subject(); err == nil && subj != "" {
		subject = &subj
	}

	if addrs, err := header.AddressList("From"); err == nil && len(addrs) > 0 {
		s := addressesToString(addrs)
		from = &s
	}

	if addrs, err := header.AddressList("Sender"); err == nil && len(addrs) > 0 {
		s := addressesToString(addrs)
		sender = &s
	}

	if addrs, err := header.AddressList("Reply-To"); err == nil && len(addrs) > 0 {
		s := addressesToString(addrs)
		replyTo = &s
	}

	if addrs, err := header.AddressList("To"); err == nil && len(addrs) > 0 {
		s := addressesToString(addrs)
		to = &s
	}

	if addrs, err := header.AddressList("Cc"); err == nil && len(addrs) > 0 {
		s := addressesToString(addrs)
		cc = &s
	}

	if addrs, err := header.AddressList("Bcc"); err == nil && len(addrs) > 0 {
		s := addressesToString(addrs)
		bcc = &s
	}

	if inReplyToVal, err := header.Text("In-Reply-To"); err == nil && inReplyToVal != "" {
		inReplyTo = &inReplyToVal
	}

	if date, err := header.Date(); err == nil {
		messageDate = &date
	}

	return
}
