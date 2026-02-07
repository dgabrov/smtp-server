package imp

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

func (s *session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	res := imap.SearchData{}

	search := ""
	if criteria != nil && len(criteria.Body) > 0 {
		search = strings.Join(criteria.Body, "%")
	}

	numSeqs, uids, err := s.srvr.SearchMessages(s.ctx, s.selected.MailboxID, search)
	if err != nil {
		return nil, err
	}

	if kind == imapserver.NumKindSeq {
		res.All = imap.SeqSetNum(numSeqs...)
	} else {
		res.All = imap.UIDSetNum(uids...)
	}

	return &res, nil
}
