package models

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
)

func TestPercentages(t *testing.T) {
	pct := NewPercentage(50)

	assert.Equal(t, "50%", pct.String())
	assert.Equal(t, 50.0, pct.InexactFloat64())
	assert.Equal(t, pct.ApplyTo(alpacadecimal.NewFromInt(100)).String(), alpacadecimal.NewFromInt(50).String())
	assert.Equal(t, pct.ApplyMarkupTo(alpacadecimal.NewFromInt(100)).String(), alpacadecimal.NewFromInt(150).String())
}
