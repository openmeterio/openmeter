# advancecharges

<!-- archie:ai-start -->

> Cobra sub-command package for charge advance operations (list customers with pending charges, advance a single customer, advance all). Guards every execution path against `internal.App.ChargesAutoAdvancer == nil` because charges are an optional, conditionally-wired feature.

## Patterns

**Nil guard for optional features** — Check `internal.App.ChargesAutoAdvancer == nil` at the start of every RunE; return a descriptive error instead of allowing a nil pointer panic. The field is nil when charges are disabled in config. (`if internal.App.ChargesAutoAdvancer == nil { return fmt.Errorf("charges are not enabled") }`)
**internal.App singleton access** — All service calls go through `internal.App.<Field>`; never construct adapters locally. (`internal.App.ChargesAutoAdvancer.ListCustomersToAdvance(cmd.Context(), ns)`)
**cmd.Context() for context propagation** — Pass `cmd.Context()` to every service call; never substitute `context.Background()`. (`internal.App.ChargesAutoAdvancer.AdvanceCharges(cmd.Context(), customer.CustomerID{Namespace: namespace, ID: args[0]})`)
**Sub-command factory functions** — Each sub-command is a `var <Name>Cmd = func() *cobra.Command` factory registered in `init()` via `Cmd.AddCommand()`. (`var AllCmd = func() *cobra.Command { cmd := &cobra.Command{Use: "all", ...}; return cmd }`)
**PersistentFlags on parent Cmd for shared filters** — --namespace is registered on the parent Cmd so all sub-commands inherit it without re-registration. (`Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace the operation should be performed")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advancecharges.go` | Defines parent Cmd plus ListCmd, CustomerCmd, AllCmd for charge advancement. CustomerCmd requires --namespace and exactly one CUSTOMER_ID arg. | Every RunE must check `ChargesAutoAdvancer != nil` first — the field is nil when billing charges are disabled in config. Missing this guard causes a panic with no user-friendly message. |

## Anti-Patterns

- Omitting the nil check on ChargesAutoAdvancer — causes panic when charges feature is disabled
- Constructing charges services locally instead of using internal.App
- Using context.Background() instead of cmd.Context()
- Registering --namespace as a persistent flag on a sub-command instead of the parent Cmd

## Decisions

- **Explicit nil check for optional ChargesAutoAdvancer field at every RunE entry** — Charges are a conditionally-wired feature; Wire produces a nil pointer when disabled. A nil guard yields a clear user-facing error instead of a runtime panic, keeping CLI UX consistent with other optional features.

## Example: Adding a new charge sub-command that requires the optional ChargesAutoAdvancer

```
var NewCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "new-op",
		RunE: func(cmd *cobra.Command, args []string) error {
			if internal.App.ChargesAutoAdvancer == nil {
				return fmt.Errorf("charges are not enabled")
			}
			if namespace == "" {
				return fmt.Errorf("namespace is required")
			}
			return internal.App.ChargesAutoAdvancer.AdvanceCharges(cmd.Context(), customer.CustomerID{
				Namespace: namespace,
				ID:        args[0],
			})
		},
// ...
```

<!-- archie:ai-end -->
