package imp

import (
	"fmt"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
)

func (s *session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	if s.selected == nil {
		return &imap.CopyData{
			UIDValidity: data.UIDValidity,
		}, nil
	}

	mbox, err := s.srvr.GetMailboxByName(s.ctx, s.userID, dest)
	if err != nil {
		return nil, err
	}

	if mbox == nil {
		return nil, fmt.Errorf("mailbox %s not found", dest)
	}

	// found mailbox, continue
	copyData, err := s.srvr.CopyMessages(s.ctx, numSet, s.selected.MailboxID, mbox.MailboxID)

	return copyData, err
}
