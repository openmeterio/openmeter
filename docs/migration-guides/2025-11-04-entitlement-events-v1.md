# Entitlement Events V1 Deprecation

This commit finally removes the V1 Entitlement Events from the codebase.
The V2 events were introduced in [85f7ec90](https://github.com/openmeterio/openmeter/commit/85f7ec9080ce9db3f8f5f363e2fa6d62270f4357) due to breaking schema changes.
At that point production of the V1 events was stopped in favor of the V2 events. **Before upgrading to this version, validate in your event processing that all V1 events are drained!**
This is especially relevant if you're upgrading from a version prior to [85f7ec90](https://github.com/openmeterio/openmeter/commit/85f7ec9080ce9db3f8f5f363e2fa6d62270f4357).
