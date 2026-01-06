package data

import "time"

type DmDomain struct {
	DomainID      string
	Name          string
	CatchAll      bool
	CatchAllLogin string
}

type DmMailbox struct {
	MailboxID string
	UserID    string
	Name      string
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
