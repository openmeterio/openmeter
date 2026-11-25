package rte_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

func TestCheckpointTimeline(t *testing.T) {
	at := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	boundary := func(offset time.Duration, value string) rte.Checkpoint {
		return rte.Checkpoint{At: at.Add(offset * time.Minute), Data: json.RawMessage(value)}
	}

	// A rebuild can replace an old payload at the same boundary. A clock moving
	// backwards can also put the calculation before the latest saved checkpoint.
	history := rte.Checkpoints{
		boundary(-1, "99"), boundary(1, "6"), boundary(-4, "1"), boundary(0, "5"),
		boundary(-1, "4"), boundary(-3, "2"), boundary(-2, "3"),
		boundary(0, `"`+strings.Repeat("x", 64<<10)+`"`), boundary(0, "invalid"),
	}

	require.Equal(t, rte.Checkpoints{
		boundary(0, "5"), boundary(-1, "99"), boundary(-2, "3"), boundary(-3, "2"),
	}, history.Candidates(at))
}

func TestCheckpointLimitUsesCompactJSON(t *testing.T) {
	at := time.Now()
	payload := json.RawMessage(`{"k":` + strings.Repeat(" ", 128) + `"` + strings.Repeat("x", (64<<10)-8) + `"}`)
	candidates := (rte.Checkpoints{{At: at, Data: payload}}).Candidates(at)
	require.Len(t, candidates, 1)
	require.Len(t, candidates[0].Data, 64<<10)
}
