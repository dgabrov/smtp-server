package imp

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/google/uuid"
)

type session struct {
	ctx              context.Context
	id               string
	srvr             srv.Servr
	userID           string
	authenticated    bool
	uidValidity      uint32
	selected         *data.DmMailbox
	lastMessageCount uint32
}

func newImapSession(server srv.Servr) imapserver.Session {
	return &session{
		id:          uuid.NewString(),
		srvr:        server,
		ctx:         context.Background(),
		uidValidity: data.UIDValidity, // will not change; this is for mailboxes, and only change when you destroy the database
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
	mbox, err := s.srvr.GetMailboxByName(s.ctx, s.userID, mailbox)
	if err != nil {
		return nil, err
	}

	if mbox == nil {
		return nil, fmt.Errorf("mailbox not found: %s", mailbox)
	}

	statusData, err := s.srvr.GetMailboxStatus(s.ctx, mbox.MailboxID)
	if err != nil {
		return nil, err
	}

	if statusData == nil {
		return nil, fmt.Errorf("mailbox not found: %s", mbox.MailboxID)
	}

	// update this
	s.selected = mbox
	s.lastMessageCount = statusData.NumMessages

	return &imap.SelectData{
		Flags:             data.AllowedFlags,
		PermanentFlags:    data.PermanentFlags,
		NumMessages:       statusData.NumMessages,
		FirstUnseenSeqNum: statusData.FirstUnseenSeqNum,
		NumRecent:         statusData.NumRecent,
		UIDNext:           imap.UID(statusData.UIDNext),
		UIDValidity:       data.UIDValidity,
		List: &imap.ListData{
			Attrs:   mbox.Attributes,
			Delim:   data.MailboxSeparatorRune,
			Mailbox: mbox.Name,
		},
		HighestModSeq: 0,
	}, nil
}

func (s *session) Create(mailbox string, options *imap.CreateOptions) error {
	// create only leaf mailboxes, if parents do not exist, will not create them automatically
	// also check if the mailbox exists

	mbox, err := s.srvr.GetMailboxByName(s.ctx, s.userID, mailbox)
	if err != nil {
		return err
	}

	if mbox != nil {
		return fmt.Errorf("mailbox already exists: %s", mailbox)
	}

	// get parent names
	items := strings.Split(mailbox, data.MailboxSeparator)
	nr := len(items)
	if nr > 1 { // if there are parent mailboxes
		nr = nr - 1

		for i := 0; i < nr; i++ {
			slc := items[0:i]

			parentName := strings.Join(slc, data.MailboxSeparator)
			parentMailbox, err := s.srvr.GetMailboxByName(s.ctx, s.userID, parentName)
			if err != nil {
				return err
			}

			if parentMailbox == nil {
				return fmt.Errorf("parent mailbox not found: %s", parentName)
			}
		}
	}

	// ok, all the parent mailboxes are accounted for, create the mailbox now
	newMailboxID := uuid.NewString()
	return s.srvr.CreateMailbox(s.ctx, s.userID, newMailboxID, mailbox)
}

func (s *session) Delete(mailbox string) error {
	mbox, err := s.srvr.GetMailboxByName(s.ctx, s.userID, mailbox)
	if err != nil {
		return err
	}

	if mbox == nil {
		slog.Info(fmt.Sprintf("mailbox not found: %s, so nothing to delete here", mailbox))
		return nil
	}

	subMailbox, err := s.srvr.GetChildMailboxes(s.ctx, s.userID, mbox.MailboxID)
	if err != nil {
		return err
	}

	if len(subMailbox) > 0 {
		return errors.New("cannot delete mailbox because it has child mailboxes")
	}

	return nil
}

func (s *session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	serv := s.srvr
	mbox, err := serv.GetMailboxByName(s.ctx, s.userID, mailbox)
	if err != nil {
		return err
	}

	if mbox == nil {
		return fmt.Errorf("mailbox not found: %s", mailbox)
	}

	// gather the child mailboxes, they need to have name modified as well
	childMailboxes, err := serv.GetChildMailboxes(s.ctx, s.userID, mbox.MailboxID)
	if err != nil {
		return err
	}

	err = serv.UpdateMailboxName(s.ctx, s.userID, mbox.MailboxID, newName)
	if err != nil {
		return err
	}

	for _, childMailbox := range childMailboxes {
		// the mailbox name will have the first characters replaced with the new one
		currentName := childMailbox.Name
		ln := len(mailbox)
		newChildName := newName + currentName[ln:]

		err = serv.UpdateMailboxName(s.ctx, s.userID, mbox.MailboxID, newChildName)
		if err != nil {
			return err
		}
	}

	return nil
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
	list, err := s.srvr.ListMailboxes(s.ctx, s.userID)
	if err != nil {
		return err
	}

	for _, mailbox := range list {
		data := imap.ListData{
			Attrs:   mailbox.Attributes,
			Delim:   data.MailboxSeparatorRune,
			Mailbox: mailbox.Name,
		}

		if !slices.Contains(data.Attrs, imap.MailboxAttrSubscribed) {
			data.Attrs = append(data.Attrs, imap.MailboxAttrSubscribed)
		}

		err = w.WriteList(&data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	servr := s.srvr
	mbox, err := servr.GetMailboxByName(s.ctx, s.userID, mailbox)
	if err != nil {
		return nil, err
	}

	mailboxID := mbox.MailboxID

	status, err := servr.GetMailboxStatus(s.ctx, mailboxID)
	if err != nil {
		return nil, err
	}

	return &imap.StatusData{
		Mailbox:     mailbox,
		NumMessages: &status.NumMessages,
		UIDNext:     imap.UID(status.UIDNext),
		UIDValidity: data.UIDValidity,
		NumUnseen:   &status.NumUnseen,
		NumDeleted:  &status.NumDeleted,
	}, nil
}

func (s *session) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	mbox, err := s.srvr.GetMailboxByName(s.ctx, s.userID, mailbox)
	if err != nil {
		return nil, err
	}

	mailboxID := mbox.MailboxID
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	uid, err := s.srvr.AppendMessage(s.ctx, mailboxID, body, options.Flags, options.Time)

	if err != nil {
		return nil, err
	}

	return &imap.AppendData{
		UID:         uid,
		UIDValidity: data.UIDValidity,
	}, nil
}

func (s *session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	if s.selected != nil {
		count, err := s.srvr.GetMessageCount(s.ctx, s.selected.MailboxID)
		if err != nil {
			return err
		}

		if count != s.lastMessageCount {
			err = w.WriteNumMessages(count)

			if err != nil {
				return err
			}

			// update last message count
			s.lastMessageCount = count
		}
	}

	return nil
}

func (s *session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			// The client sent "DONE" or disconnected.
			// Exit the function to stop idling.
			return nil

		case <-ticker.C:
			// Time to check the DB!
			// We pass 'true' for allowExpunge because the IMAP spec
			// allows sending updates during IDLE.
			if err := s.Poll(w, true); err != nil {
				return err
			}
		}
	}

}

func (s *session) Unselect() error {
	s.selected = nil
	s.lastMessageCount = 0

	return nil
}

func (s *session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	if s.selected == nil {
		slog.Info("no mailbox selected, we cannot expunge")

		return nil
	}

	seqHolders, err := s.srvr.GetExpungeInformation(s.ctx, uids, s.selected.MailboxID)
	if err != nil {
		return err
	}

	// and now for each of it delete it and then pass the number, but make sure you operate from maximum to minimum
	slices.SortFunc(seqHolders, func(first *data.SeqHolder, second *data.SeqHolder) int {
		return cmp.Compare(second.NumSeq, first.NumSeq)
	})

	for _, seqHolder := range seqHolders {
		numSeq := seqHolder.NumSeq
		id := seqHolder.ID

		err := s.srvr.DeleteMessage(s.ctx, s.selected.MailboxID, id)
		if err != nil {
			return err
		}

		err = w.WriteExpunge(numSeq)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	// TODO
	return nil, errors.New("search not implemented")
}

func (s *session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	// TODO
	return errors.New("fetch not implemented")
}

func (s *session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	if s.selected == nil {
		return nil
	}

	items, err := s.srvr.GetFilteredPositionalData(s.ctx, s.selected.MailboxID, numSet)
	if err != nil {
		return err
	}

	var newFlags []imap.Flag

	for _, item := range items {
		switch flags.Op {
		case imap.StoreFlagsSet:
			newFlags, err = s.srvr.SetMessageFlags(s.ctx, item.MessageID, flags.Flags)

			if err != nil {
				return err
			}
		case imap.StoreFlagsDel:
			newFlags, err = s.srvr.DeleteMessageFlags(s.ctx, item.MessageID, flags.Flags)

			if err != nil {
				return err
			}
		case imap.StoreFlagsAdd:
			newFlags, err = s.srvr.AddMessageFlags(s.ctx, item.MessageID, flags.Flags)

			if err != nil {
				return err
			}
		}

		// proceed
		responseWriter := w.CreateMessage(item.SeqNum)
		responseWriter.WriteFlags(newFlags)
		responseWriter.WriteUID(item.UID)

		err = responseWriter.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
