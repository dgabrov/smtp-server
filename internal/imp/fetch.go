package imp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-message/textproto"
)

func (s *session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	if s.selected == nil {
		return errors.New("there is no mailbox selected, not sure why fetch was invoked")
	}

	messages, err := s.srvr.GetStrippedMessages(s.ctx, numSet, s.selected.MailboxID)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("fetch, mailbox selected: %s, numset: %s, found %d messages", s.selected.MailboxID, numSet.String(), len(messages)))

	for _, msg := range messages {
		messageID := msg.MessageID

		messageBody, err := s.srvr.GetMessageBody(s.ctx, messageID)
		markSeen, err := processMessage(w, *msg, messageBody, options)

		if err != nil {
			return err
		}

		if markSeen {
			err = s.srvr.MarkMessageAsSeen(s.ctx, messageID)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func processMessage(w *imapserver.FetchWriter, msg data.DmStrippedMessage, messageBody []byte, options *imap.FetchOptions) (bool, error) {
	mg := w.CreateMessage(msg.SeqNum)
	defer mg.Close()

	bytesReader := bytes.NewReader(messageBody)
	r := bufio.NewReader(bytesReader)
	header, err := textproto.ReadHeader(r)
	if err != nil {
		return false, err
	}

	if options.Envelope {
		envelope := imapserver.ExtractEnvelope(header)

		mg.WriteEnvelope(envelope)
	}

	if options.UID {
		mg.WriteUID(msg.UID)
	}

	if options.RFC822Size {
		messageLength := len(messageBody)
		in64MessageLength := int64(messageLength)

		mg.WriteRFC822Size(in64MessageLength)
	}

	if options.InternalDate {
		mg.WriteInternalDate(msg.InternalDate.UTC())
	}

	if options.Flags {
		mg.WriteFlags(msg.Flags)
	}

	if options.BodyStructure != nil {
		// send the body structure
		bodyReader := bytes.NewReader(messageBody)
		bodyStructure := imapserver.ExtractBodyStructure(bodyReader)

		mg.WriteBodyStructure(bodyStructure)
	}

	// body section - very difficult to process
	markSeen := false

	for _, bodySection := range options.BodySection {
		if !bodySection.Peek {
			markSeen = true
		}

		sectionBytes := imapserver.ExtractBodySection(bytes.NewReader(messageBody), bodySection)
		ln := len(sectionBytes)

		wc := mg.WriteBodySection(bodySection, int64(ln))
		_, err = io.Copy(wc, bytes.NewReader(sectionBytes))

		_ = wc.Close()

		if err != nil {
			return false, err
		}
	}

	return markSeen, nil
}
