package plan

import (
	"errors"
	"fmt"
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
			ExpectedError error
		}{
			{
				Name: "valid",
				Price: FlatPrice{
					PriceMeta: PriceMeta{
						Type: FlatPriceType,
					},
					Amount:      decimal.NewFromInt(1000),
					PaymentTerm: lo.ToPtr(InArrearsPaymentTerm),
				},
				ExpectedError: nil,
			},
			{
				Name: "invalid",
				Price: FlatPrice{
					PriceMeta: PriceMeta{
						Type: FlatPriceType,
					},
					Amount:      decimal.NewFromInt(-1000),
					PaymentTerm: lo.ToPtr(PaymentTermType("invalid")),
				},
				ExpectedError: errors.Join([]error{
					errors.New("amount must not be negative"),
					fmt.Errorf("invalid payment term: %s", "invalid"),
				}...),
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Price.Validate()
				assert.Equal(t, test.ExpectedError, err)
			})
		}
	})
}

func TestUnitPrice(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			Price         UnitPrice
			ExpectedError error
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
				ExpectedError: nil,
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
				ExpectedError: nil,
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
				ExpectedError: nil,
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
				ExpectedError: errors.Join([]error{
					errors.New("amount must not be negative"),
					errors.New("minimum amount must not be negative"),
					errors.New("maximum amount must not be negative"),
					errors.New("minimum amount must not be greater than maximum amount"),
				}...),
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Price.Validate()
				assert.Equal(t, test.ExpectedError, err)
			})
		}
	})
}

func TestTieredPrice(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			Price         TieredPrice
			ExpectedError error
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
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(5),
								PaymentTerm: lo.ToPtr(InAdvancePaymentTerm),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(5),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(100)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(3),
								PaymentTerm: lo.ToPtr(InArrearsPaymentTerm),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(1),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(2500)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
							},
						},
					},
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
				},
				ExpectedError: nil,
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
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(5),
								PaymentTerm: lo.ToPtr(InAdvancePaymentTerm),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(5),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(100)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(3),
								PaymentTerm: lo.ToPtr(InArrearsPaymentTerm),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(1),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(2500)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
							},
						},
					},
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
					MaximumAmount: nil,
				},
				ExpectedError: nil,
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
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(5),
								PaymentTerm: lo.ToPtr(InAdvancePaymentTerm),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(5),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(100)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(2500)),
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(3),
								PaymentTerm: lo.ToPtr(InArrearsPaymentTerm),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(1),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(2500)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(5000)),
							},
						},
					},
					MinimumAmount: nil,
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
				},
				ExpectedError: nil,
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
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(-5),
								PaymentTerm: lo.ToPtr(PaymentTermType("invalid")),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(-5),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(-100)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
							},
						},
						{
							UpToAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
							FlatPrice: &FlatPrice{
								PriceMeta: PriceMeta{
									Type: FlatPriceType,
								},
								Amount:      decimal.NewFromInt(-3),
								PaymentTerm: lo.ToPtr(PaymentTermType("invalid")),
							},
							UnitPrice: &UnitPrice{
								PriceMeta: PriceMeta{
									Type: UnitPriceType,
								},
								Amount:        decimal.NewFromInt(-1),
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(-2500)),
								MaximumAmount: lo.ToPtr(decimal.NewFromInt(-5000)),
							},
						},
					},
					MinimumAmount: lo.ToPtr(decimal.NewFromInt(-1000)),
					MaximumAmount: lo.ToPtr(decimal.NewFromInt(-5000)),
				},
				ExpectedError: errors.Join([]error{
					errors.New("invalid tiered price mode: invalid"),
					fmt.Errorf("invalid price tier: %w", errors.Join([]error{
						errors.New("up-to-amount must not be negative"),
						fmt.Errorf("invalid flat price: %w", errors.Join([]error{
							errors.New("amount must not be negative"),
							errors.New("invalid payment term: invalid"),
						}...)),
						fmt.Errorf("invalid unit price: %w", errors.Join([]error{
							errors.New("amount must not be negative"),
							errors.New("minimum amount must not be negative"),
							errors.New("maximum amount must not be negative"),
							errors.New("minimum amount must not be greater than maximum amount"),
						}...)),
					}...)),
					errors.New("multiple price tiers with same up-to-amount are not allowed"),
					errors.New("minimum amount must not be negative"),
					errors.New("maximum amount must not be negative"),
					errors.New("minimum amount must not be greater than maximum amount"),
				}...),
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Price.Validate()
				assert.Equal(t, test.ExpectedError, err)
			})
		}
	})
}
