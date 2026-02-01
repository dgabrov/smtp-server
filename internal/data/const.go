package data

import "github.com/emersion/go-imap/v2"

const (
	MailboxSeparatorRune = '.'
	MailboxSeparator     = string(MailboxSeparatorRune)
	UIDValidity          = 1
)

var (
	AllowedFlags   = []imap.Flag{imap.FlagSeen, imap.FlagAnswered, imap.FlagFlagged, imap.FlagDeleted, imap.FlagDraft}
	PermanentFlags = []imap.Flag{imap.FlagSeen, imap.FlagAnswered, imap.FlagFlagged, imap.FlagDeleted, imap.FlagDraft, imap.FlagWildcard}
)
