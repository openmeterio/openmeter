package plan

import (
	"encoding/json"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrice_JSON(t *testing.T) {
	tests := []struct {
		Name          string
		Price         Price
		ExpectedError bool
	}{
		{
			Name: "Flat",
			Price: NewPriceFrom(FlatPrice{
				Amount:      decimal.NewFromInt(1000),
				PaymentTerm: InAdvancePaymentTerm,
			}),
		},
		{
			Name: "Unit",
			Price: NewPriceFrom(UnitPrice{
				Amount:        decimal.NewFromInt(1000),
				MinimumAmount: lo.ToPtr(decimal.NewFromInt(10)),
				MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
			}),
		},
		{
			Name: "Tiered",
			Price: NewPriceFrom(TieredPrice{
				Mode: VolumeTieredPrice,
				Tiers: []PriceTier{
					{
						UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
						FlatPrice: &PriceTierFlatPrice{
							Amount: decimal.NewFromInt(1000),
						},
						UnitPrice: &PriceTierUnitPrice{
							Amount: decimal.NewFromInt(5),
						},
					},
					{
						UpToAmount: nil,
						FlatPrice: &PriceTierFlatPrice{
							Amount: decimal.NewFromInt(1500),
						},
						UnitPrice: &PriceTierUnitPrice{
							Amount: decimal.NewFromInt(1),
						},
					},
				},
				MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
				MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.Price)
			require.NoError(t, err)

			t.Logf("Serialized Price: %s", string(b))

			d := Price{}
			err = json.Unmarshal(b, &d)
			require.NoError(t, err)

			assert.Equal(t, test.Price, d)
		})
	}
}

func TestFlatPrice(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			Price         FlatPrice
			ExpectedError bool
		}{
			{
				Name: "valid",
				Price: FlatPrice{
					Amount:      decimal.NewFromInt(1000),
					PaymentTerm: InArrearsPaymentTerm,
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				Price: FlatPrice{
					Amount:      decimal.NewFromInt(-1000),
					PaymentTerm: PaymentTermType("invalid"),
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Price.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestUnitPrice(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			Price         UnitPrice
			ExpectedError bool
		}{
			{
				Name: "valid with min,max",
				Price: UnitPrice{
					Amount:        decimal.NewFromInt(1000),
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
				},
				ExpectedError: false,
			},
			{
				Name: "valid with min only",
				Price: UnitPrice{
					Amount:        decimal.NewFromInt(1000),
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: nil,
				},
				ExpectedError: false,
			},
			{
				Name: "valid with max only",
				Price: UnitPrice{
					Amount:        decimal.NewFromInt(1000),
					MinimumAmount: nil,
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				Price: UnitPrice{
					Amount:        decimal.NewFromInt(-1000),
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(-2000)),
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Price.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestTieredPrice(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			Price         TieredPrice
			ExpectedError bool
		}{
			{
				Name: "valid with min,max",
				Price: TieredPrice{
					Mode: VolumeTieredPrice,
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(5),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(3),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(1),
							},
						},
					},
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
				},
				ExpectedError: false,
			},
			{
				Name: "valid with min only",
				Price: TieredPrice{
					Mode: VolumeTieredPrice,
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(5),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(3),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(1),
							},
						},
					},
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: nil,
				},
				ExpectedError: false,
			},
			{
				Name: "valid with max only",
				Price: TieredPrice{
					Mode: VolumeTieredPrice,
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(5),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(3),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(1),
							},
						},
					},
					MinimumAmount: nil,
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				Price: TieredPrice{
					Mode: TieredPriceMode("invalid"),
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(-5),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(-5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
							FlatPrice: &PriceTierFlatPrice{
								Amount: decimal.NewFromInt(-3),
							},
							UnitPrice: &PriceTierUnitPrice{
								Amount: decimal.NewFromInt(-1),
							},
						},
					},
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(-5000)),
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Price.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
