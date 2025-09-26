package grant_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEffectivePeriod(t *testing.T) {
	now := clock.Now()

	t.Run("base case", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.ExpiresAt, p.To)
	})

	t.Run("no expiration", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Nil(t, p.To)
	})

	t.Run("deleted ineffectual", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			ManagedModel: models.ManagedModel{
				DeletedAt: lo.ToPtr(now.Add(time.Minute + time.Hour)), // 1H1M
			},
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.ExpiresAt, p.To)
	})

	t.Run("deleted before expiration", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			ManagedModel: models.ManagedModel{
				DeletedAt: lo.ToPtr(now.Add(time.Minute)), // 1M
			},
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.DeletedAt, p.To)
	})

	t.Run("no expiration deleted later", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ManagedModel: models.ManagedModel{
				DeletedAt: lo.ToPtr(now.Add(time.Minute)), // 1M
			},
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.DeletedAt, p.To)
	})

	t.Run("deleted before effective", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			ManagedModel: models.ManagedModel{
				DeletedAt: lo.ToPtr(now.Add(-time.Minute)), // -1M
			},
		}

		p := g.GetEffectivePeriod()

		// Grants that never activate have a 0 length period starting and ending at the effective date
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.EffectiveAt, lo.FromPtr(p.To))
	})

	t.Run("no expiration deleted before effective", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ManagedModel: models.ManagedModel{
				DeletedAt: lo.ToPtr(now.Add(-time.Minute)), // -1M
			},
		}

		p := g.GetEffectivePeriod()

		// Grants that never activate have a 0 length period starting and ending at the effective date
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.EffectiveAt, lo.FromPtr(p.To))
	})

	t.Run("voided ineffectual", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			VoidedAt:    lo.ToPtr(now.Add(time.Minute + time.Hour)), // 1H1M
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.ExpiresAt, p.To)
	})

	t.Run("voided before expiration", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			VoidedAt:    lo.ToPtr(now.Add(time.Minute)), // 1M
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.VoidedAt, p.To)
	})

	t.Run("no expiration voided later", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			VoidedAt:    lo.ToPtr(now.Add(time.Minute)), // 1M
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.VoidedAt, p.To)
	})

	t.Run("voided before effective", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			VoidedAt:    lo.ToPtr(now.Add(-time.Minute)), // -1M
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.EffectiveAt, lo.FromPtr(p.To))
	})

	t.Run("no expiration voided before effective", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			VoidedAt:    lo.ToPtr(now.Add(-time.Minute)), // -1M
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.EffectiveAt, lo.FromPtr(p.To))
	})

	t.Run("voided and deleted", func(t *testing.T) {
		g := grant.Grant{
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			VoidedAt:    lo.ToPtr(now.Add(time.Minute)), // 1M
			ManagedModel: models.ManagedModel{
				DeletedAt: lo.ToPtr(now.Add(time.Minute + time.Hour)), // 1H1M
			},
		}

		p := g.GetEffectivePeriod()
		assert.Equal(t, g.EffectiveAt, p.From)
		assert.Equal(t, g.VoidedAt, p.To)
	})
}

func TestIsActiveAt(t *testing.T) {
	now := clock.Now()

	tt := []struct {
		name string
		g    grant.Grant
		at   time.Time
		want bool
	}{
		{
			name: "base case",
			g: grant.Grant{
				EffectiveAt: now,
				ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			},
			at:   now,
			want: true,
		},
		{
			name: "not active",
			g: grant.Grant{
				EffectiveAt: now,
				ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
			},
			at:   now.Add(time.Hour),
			want: false,
		},
		{
			name: "voided",
			g: grant.Grant{
				EffectiveAt: now,
				ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
				VoidedAt:    lo.ToPtr(now.Add(time.Minute)), // 1M
			},
			at:   now.Add(time.Minute),
			want: false,
		},
		{
			name: "deleted",
			g: grant.Grant{
				EffectiveAt: now,
				ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
				ManagedModel: models.ManagedModel{
					DeletedAt: lo.ToPtr(now.Add(time.Minute)), // 1M
				},
			},
			at:   now.Add(time.Minute),
			want: false,
		},
		{
			name: "voided and deleted",
			g: grant.Grant{
				EffectiveAt: now,
				ExpiresAt:   lo.ToPtr(now.Add(time.Hour)),
				VoidedAt:    lo.ToPtr(now.Add(time.Minute)), // 1M
				ManagedModel: models.ManagedModel{
					DeletedAt: lo.ToPtr(now.Add(time.Minute + time.Hour)), // 1H1M
				},
			},
			at:   now.Add(time.Minute),
			want: false,
		},
		{
			name: "0 length",
			g: grant.Grant{
				EffectiveAt: now,
				ExpiresAt:   lo.ToPtr(now),
			},
			at:   now,
			want: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.g.ActiveAt(tc.at))
		})
	}
}
