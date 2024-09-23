package billing_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProfileTestSuite struct {
	BaseSuite
}

func TestProfile(t *testing.T) {
	suite.Run(t, new(ProfileTestSuite))
}
