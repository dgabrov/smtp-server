package queue

import (
	"bytes"
	"crypto/rsa"

	"github.com/emersion/go-msgauth/dkim"
)

func signMessage(rawMsg []byte, privKey *rsa.PrivateKey) ([]byte, error) {
	options := &dkim.SignOptions{
		Domain:   "yourdomain.com",
		Selector: "default", // <--- Must match your DNS record exactly
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
