# currencyx

`currencyx` contains OpenMeter's shared currency primitives. It keeps fiat
behavior compatible with GOBL/ISO currency definitions while allowing product
and ledger code to pass configured custom currencies through the same
`Currency` interface.

## Fiat Currency

Use `Code` directly for fiat currencies. Fiat precision comes from the GOBL
currency definition and fiat rounding remains half-away-from-zero.

```go
calculator, err := currencyx.Code("USD").Calculator()
if err != nil {
    return err
}

amount := calculator.RoundToPrecision(alpacadecimal.RequireFromString("1.235"))
// amount == 1.24
```

## Custom Currency

Use `NewCustomCurrency` when the currency is not a known fiat code. Custom
currencies carry explicit precision and default to bankers rounding
(half-even). Use `NewCustomCurrencyWithRounding` to opt into half-away-from-zero.

```go
credits, err := currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), 6)
if err != nil {
    return err
}

calculator, err := currencyx.NewCalculator(credits)
if err != nil {
    return err
}

amount := calculator.RoundToPrecision(alpacadecimal.RequireFromString("1.2345678"))
// amount == 1.234568
```

## Allocation

Allocation helpers use the calculator's precision and distribute residual units
with a deterministic largest-remainder method. Provide a `CompareKey` function
when equal remainders need a stable domain-specific tie-breaker.
