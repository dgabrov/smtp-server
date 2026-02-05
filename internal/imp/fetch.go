package imp

import (
	"bufio"
	"bytes"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-message/textproto"
)

func (s *session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	messages, err := getMessageData(numSet)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		err = processMessage(w, msg, options)
	}

	return nil
}

func processMessage(w *imapserver.FetchWriter, msg MessageData, options *imap.FetchOptions) error {
	mg := w.CreateMessage(msg.NumSeq)
	defer mg.Close()

	messageBody, flags, err := getMessageBody(msg.MessageID)
	if err != nil {
		return err
	}

	bytesReader := bytes.NewReader(messageBody)
	r := bufio.NewReader(bytesReader)
	header, err := textproto.ReadHeader(r)
	if err != nil {
		return err
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
		mg.WriteFlags(flags)
	}

	if options.BodyStructure != nil {
		// send the body structure
		bodyReader := bytes.NewReader(messageBody)
		bodyStructure := imapserver.ExtractBodyStructure(bodyReader)

		mg.WriteBodyStructure(bodyStructure)
	}
	// not nil and contianing things
	if len(options.BodySection) > 0 {
// 		for _, section := range options.BodySection {
//
// 		}
	}

	// take all the values
	return nil
}

func getMessageBody(id string) ([]byte, []imap.Flag, error) {
	return nil, nil, nil
}

/*
	BodySection       []*FetchItemBodySection
*/

func getMessageData(numSet imap.NumSet) ([]MessageData, error) {
	// returns a set of items that contain this
	return nil, nil
}

type MessageData struct {
	MessageID    string
	NumSeq       uint32
	UID          imap.UID
	InternalDate time.Time
}
