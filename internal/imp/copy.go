package imp

import (
	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
)

func (s *session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	if s.selected == nil {
		return &imap.CopyData{
			UIDValidity: data.UIDValidity,
		}, nil
	}

// 	var sourceUids []imap.UID
// 	var destinationUids []imap.UID

// 	_ := s.selected.MailboxID

	// TODO this is not at all implemented...

// 	return &imap.CopyData{
// 		UIDValidity: data.UIDValidity,
// 		SourceUIDs:  sourceUids,
// 		DestUIDs:    destinationUids,
// 	}, nil
    return nil, nil
}
