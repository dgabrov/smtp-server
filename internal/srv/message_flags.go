package srv

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/emersion/go-imap/v2"
)

func (s *server) SetMessageFlags(ctx context.Context, messageID string, flags []imap.Flag) ([]imap.Flag, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = setMessageFlags(ctx, tx, messageID, flags)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return flags, nil
}

func (s *server) DeleteMessageFlags(ctx context.Context, messageID string, flags []imap.Flag) ([]imap.Flag, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	currentFlags, err := getCurrentMessageFlags(ctx, tx, messageID)
	if err != nil {
		return nil, err
	}

	var newFlags []imap.Flag
	for _, flag := range currentFlags {
		if !slices.Contains(flags, flag) {
			newFlags = append(newFlags, flag)
		}
	}

	err = setMessageFlags(ctx, tx, messageID, newFlags)

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return newFlags, nil
}

func (s *server) AddMessageFlags(ctx context.Context, messageID string, flags []imap.Flag) ([]imap.Flag, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	currentFlags, err := getCurrentMessageFlags(ctx, tx, messageID)
	if err != nil {
		return nil, err
	}

	for _, flag := range flags {
		if !slices.Contains(currentFlags, flag) {
			currentFlags = append(currentFlags, flag)
		}
	}

	err = setMessageFlags(ctx, tx, messageID, currentFlags)

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return currentFlags, nil
}

func getCurrentMessageFlags(ctx context.Context, tx *sql.Tx, messageID string) ([]imap.Flag, error) {
	qr := "select flag_seen, flag_answered, flag_flagged, flag_deleted, flag_draft from message where message_id = ?"
	rs, err := tx.QueryContext(ctx, qr, messageID)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var flagSeen string
	var flagAnswered string
	var flagFlagged string
	var flagDeleted string
	var flagDraft string

	if rs.Next() {
		err = rs.Scan(&flagSeen, &flagAnswered, &flagFlagged, &flagDeleted, &flagDraft)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("cannot find message with id: %s", messageID)
	}

	var res []imap.Flag
	res = processMessageFlag(res, flagSeen, imap.FlagSeen)
	res = processMessageFlag(res, flagAnswered, imap.FlagAnswered)
	res = processMessageFlag(res, flagFlagged, imap.FlagFlagged)
	res = processMessageFlag(res, flagDeleted, imap.FlagDeleted)
	res = processMessageFlag(res, flagDraft, imap.FlagDraft)

	return res, nil
}

func processMessageFlag(res []imap.Flag, strValue string, flag imap.Flag) []imap.Flag {
	if strValue == "Y" {
		res = append(res, flag)
	}

	return res
}

func setMessageFlags(ctx context.Context, tx *sql.Tx, messageID string, flags []imap.Flag) error {
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

	if slices.Contains(flags, imap.FlagDraft) {
		flagDraft = "Y"
	}

	qr := "update message set flag_seen = ?, flag_answered = ?, flag_flagged = ?, flag_deleted = ?, flag_draft = ? where message_id = ?"
	_, err := tx.ExecContext(ctx, qr, flagSeen, flagAnswered, flagFlagged, flagDeleted, flagDraft, messageID)

	return err
}
