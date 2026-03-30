package taxcode

import "github.com/samber/lo"

// DefaultTaxCodeSeed holds the minimal inputs needed to seed one well-known tax code.
type DefaultTaxCodeSeed struct {
	Key         string
	Name        string
	Description *string
	StripeCode  string
}

// DefaultStripeTaxCodes is a curated list of commonly used Stripe tax codes for
// software and SaaS businesses. These are seeded at startup so users can browse
// and select from known codes without having to look up Stripe-specific identifiers.
//
// Keys follow the same convention used by GetOrCreateByAppMapping: "stripe_<code>".
// This ensures that if a code was already auto-created by the billing flow, the
// seed CreateTaxCode call will hit the unique constraint and be silently skipped.
var DefaultStripeTaxCodes = []DefaultTaxCodeSeed{
	{
		Key:         "stripe_txcd_00000000",
		Name:        "Non-taxable",
		Description: lo.ToPtr("Use for products or services that are explicitly exempt from tax."),
		StripeCode:  "txcd_00000000",
	},
	{
		Key:         "stripe_txcd_10103001",
		Name:        "SaaS - Business Use",
		Description: lo.ToPtr("Software as a Service intended for business use."),
		StripeCode:  "txcd_10103001",
	},
	{
		Key:         "stripe_txcd_10101000",
		Name:        "Infrastructure as a Service (IaaS)",
		Description: lo.ToPtr("Cloud infrastructure services such as compute, storage, and networking."),
		StripeCode:  "txcd_10101000",
	},
	{
		Key:         "stripe_txcd_10102000",
		Name:        "Platform as a Service (PaaS)",
		Description: lo.ToPtr("Cloud platform services providing a managed environment for building and deploying applications."),
		StripeCode:  "txcd_10102000",
	},
	{
		Key:         "stripe_txcd_10000000",
		Name:        "Digital Goods - General",
		Description: lo.ToPtr("General category for digital goods and electronically supplied services. Use when no more specific code applies."),
		StripeCode:  "txcd_10000000",
	},
	{
		Key:         "stripe_txcd_20030000",
		Name:        "Professional Services",
		Description: lo.ToPtr("Consulting, advisory, and other professional services."),
		StripeCode:  "txcd_20030000",
	},
}
