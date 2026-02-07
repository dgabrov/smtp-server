package data

import (
	"time"

	"github.com/emersion/go-imap/v2"
)

type DmDomain struct {
	DomainID      string
	Name          string
	CatchAll      bool
	CatchAllLogin string
}

type DmMailbox struct {
	MailboxID  string
	UserID     string
	Name       string
	Attributes []imap.MailboxAttr
}

type DmMailboxStatus struct {
	NumMessages       uint32
	FirstUnseenSeqNum uint32
	NumRecent         uint32
	UIDNext           uint32
	UIDValidity       uint32
	NumDeleted        uint32
	NumUnseen         uint32
}

type DmUser struct {
	UserID   string
	DomainID string
	Login    string
	Password string
}

type DmMessage struct {
	MessageID string
	MailboxID string
	Body      string
	Flags     []imap.Flag
}

type DmStrippedMessage struct {
	MessageID    string
	MailboxID    string
	UID          imap.UID
	SeqNum       uint32
	InternalDate time.Time
	Flags        []imap.Flag
}

type DmPositionalMessage struct {
	MessageID string
	UID       imap.UID
	SeqNum    uint32
}

type SeqHolder struct {
	ID     string
	UID    uint32
	NumSeq uint32
}

type DmQueue struct {
	QueueID string
	From    string
	Body    string
}

type DmQueueRecipient struct {
	QueueRecipientID string
	QueueID          string
	ToAddr           string
	Attempts         int
	LastAttemptedDt  time.Time
	Success          bool
}
