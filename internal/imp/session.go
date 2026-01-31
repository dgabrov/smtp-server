package imp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/google/uuid"
)

type session struct {
	ctx           context.Context
	id            string
	srvr          srv.Servr
	userID        string
	authenticated bool
	uidValidity   uint32
}

func newImapSession(server srv.Servr) imapserver.Session {
	return &session{
		id:          uuid.NewString(),
		srvr:        server,
		ctx:         context.Background(),
		uidValidity: 1, // will not change; this is for mailboxes, and only change when you destroy the database
	}
}

func (s *session) Close() error {
	slog.Info(fmt.Sprintf("Closing session: %s", s.id))

	return nil
}

func (s *session) Login(username, password string) error {
	err := s.srvr.Authenticate(s.ctx, username, password)
	if err != nil {
		return err
	}

	s.authenticated = true

	// search for some user and try to see if you find it, should be there
	userID, err := s.srvr.GetUserID(s.ctx, username)
	if err != nil {
		return err
	}

	s.userID = userID

	return nil
}

func (s *session) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	//TODO implement me
	panic("implement me")
}

func (s *session) Create(mailbox string, options *imap.CreateOptions) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Delete(mailbox string) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Subscribe(mailbox string) error {
	slog.Info(fmt.Sprintf("Subscribing to %s", mailbox))

	return nil
}

func (s *session) Unsubscribe(mailbox string) error {
	slog.Info(fmt.Sprintf("Unsubscribing from %s", mailbox))

	return nil
}

func (s *session) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {

	//TODO implement me -- always return \Subscribed flag!!!
	panic("implement me")
}

func (s *session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	//TODO implement me
	panic("implement me")
}

func (s *session) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	//TODO implement me
	panic("implement me")
}

func (s *session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Unselect() error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	//TODO implement me
	panic("implement me")
}

func (s *session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	//TODO implement me
	panic("implement me")
}

func (s *session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	//TODO implement me
	panic("implement me")
}
