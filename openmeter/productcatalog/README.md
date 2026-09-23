# Product catalog

Plan default billing cadences use the explicit allowlist in `ErrPlanBillingCadenceAllowedValues`. Rate-card validation requires a billing cadence of at least 24 hours. `P1D` and `PT24H` both meet that minimum; they remain distinct periods across daylight-saving changes. A flat-fee rate card may omit its cadence for a one-time charge.

The minimum is a warning during draft plan and add-on authoring when non-critical issues are allowed. Publishing either resource and creating a subscription still reject it. A new plan draft can be copied from a plan with warning-only validation issues; those issues must be fixed before publishing. Copying a legacy add-on remains blocked. Subscription cancellation permits existing shorter cadences only when they are the sole validation issues. No stored cadence is automatically rewritten.
