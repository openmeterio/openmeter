package webhook

import (
	"crypto/rand"
	"encoding/base64"
)

const DefaultSigningSecretSize = 32

func NewSigningSecret(size int) (string, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return "whsec_" + base64.StdEncoding.EncodeToString(b), nil
}

func NewSigningSecretWithDefaultSize() (string, error) {
	return NewSigningSecret(DefaultSigningSecretSize)
}
