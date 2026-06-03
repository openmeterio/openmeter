# Enforcement: services (1 rule)

Topic file. Loaded on demand when an agent works on something in the `services` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pitfalls (block)

### `pf-006a-worker-namespace-precondition` — Worker binaries must fail-fast assert that the default namespace and its handler-provisioned subsystems exist before running namespace-scoped provisioning. Only cmd/server registers namespace handlers; workers must not assume the default namespace is already provisioned.

*source: `deep_scan`*

**Why:** Pitfall pf_0006a: namespace handlers (Ledger, KafkaIngest) are registered only by cmd/server, while worker binaries (cmd/billing-worker/main.go:86 calls EnsureBusinessAccounts/SandboxProvisioner) perform namespace-scoped provisioning inline assuming the default namespace already exists, creating an unenforced cross-binary boot-order contract. namespace.go:105-118 createNamespace fans out via errors.Join (no short-circuit) and requires RegisterHandler before CreateDefaultNamespace — enforced nowhere; deploy/charts/openmeter has no init-container ordering guarantee.

**Example:**

```
// In a worker before namespace-scoped provisioning:
if _, err := namespaceManager.GetDefaultNamespace(ctx); err != nil {
    return fmt.Errorf("default namespace not provisioned by cmd/server yet: %w", err)
}
return app.EnsureBusinessAccounts(ctx, ns)
```

**Path glob:** `cmd/billing-worker/main.go`, `cmd/balance-worker/main.go`, `cmd/sink-worker/main.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "EnsureBusinessAccounts|SandboxProvisioner|CreateNamespace"
    ],
    "must_not_match": [
      "GetDefaultNamespace"
    ]
  }
]
```

</details>
