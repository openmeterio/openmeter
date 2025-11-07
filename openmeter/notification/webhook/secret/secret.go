package secret

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/openmeterio/openmeter/pkg/models"
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

const (
	SigningSecretPrefix = "whsec_"
)

func ValidateSigningSecret(secret string) error {
	var errs []error

	s, _ := strings.CutPrefix(secret, SigningSecretPrefix)
	if len(s) < 32 || len(s) > 100 {
		errs = append(errs, errors.New("secret length must be between 32 to 100 chars without the optional prefix"))
	}

	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		errs = append(errs, errors.New("invalid base64 string"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
