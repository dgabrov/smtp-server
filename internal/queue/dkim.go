package queue

import (
	"bytes"
	"crypto/rsa"

	"github.com/emersion/go-msgauth/dkim"
)

func signMessage(rawMsg []byte, privKey *rsa.PrivateKey, mailDomain string, selector string) ([]byte, error) {
	options := &dkim.SignOptions{
		Domain:   mailDomain,
		Selector: selector,
		Signer:   privKey,
		// Using Relaxed for both ensures slight header/body
		// modifications by relays won't break the signature.
		HeaderCanonicalization: dkim.CanonicalizationRelaxed,
		BodyCanonicalization:   dkim.CanonicalizationRelaxed,
	}

	var b bytes.Buffer
	// dkim.Sign takes an io.Reader (the original email)
	// and writes the signed version to the buffer.
	if err := dkim.Sign(&b, bytes.NewReader(rawMsg), options); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
