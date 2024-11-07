package plan

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

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
					PriceMeta: PriceMeta{
						Type: FlatPriceType,
					},
					Amount:      decimal.NewFromInt(1000),
					PaymentTerm: InArrearsPaymentTerm,
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				Price: FlatPrice{
					PriceMeta: PriceMeta{
						Type: FlatPriceType,
					},
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
					PriceMeta: PriceMeta{
						Type: UnitPriceType,
					},
					Amount:        decimal.NewFromInt(1000),
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
				},
				ExpectedError: false,
			},
			{
				Name: "valid with min only",
				Price: UnitPrice{
					PriceMeta: PriceMeta{
						Type: UnitPriceType,
					},
					Amount:        decimal.NewFromInt(1000),
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: nil,
				},
				ExpectedError: false,
			},
			{
				Name: "valid with max only",
				Price: UnitPrice{
					PriceMeta: PriceMeta{
						Type: UnitPriceType,
					},
					Amount:        decimal.NewFromInt(1000),
					MinimumAmount: nil,
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				Price: UnitPrice{
					PriceMeta: PriceMeta{
						Type: UnitPriceType,
					},
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
					PriceMeta: PriceMeta{
						Type: TieredPriceType,
					},
					Mode: VolumeTieredPrice,
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(5),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount: decimal.NewFromInt(5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(3),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
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
					PriceMeta: PriceMeta{
						Type: TieredPriceType,
					},
					Mode: VolumeTieredPrice,
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(5),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount: decimal.NewFromInt(5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(3),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
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
					PriceMeta: PriceMeta{
						Type: TieredPriceType,
					},
					Mode: VolumeTieredPrice,
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(5),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount: decimal.NewFromInt(5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(3),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
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
					PriceMeta: PriceMeta{
						Type: TieredPriceType,
					},
					Mode: TieredPriceMode("invalid"),
					Tiers: []PriceTier{
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(-5),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount: decimal.NewFromInt(-5),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
							FlatPrice: &PriceTierFlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount: decimal.NewFromInt(-3),
							},
							UnitPrice: &PriceTierUnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
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
