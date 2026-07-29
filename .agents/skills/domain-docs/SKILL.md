---
name: domain-docs
description: Locate and maintain OpenMeter's package-level domain documentation. Use for code reviews, planning, implementation, debugging, or explanations that depend on product behavior, domain ownership, or cross-domain contracts, and when creating or editing domain READMEs. Do not use for mechanical changes with no domain behavior.
---

# Domain docs

Route domain work to the relevant package documentation and keep that
documentation accurate, useful to humans, and small enough to read.

## Find the relevant docs

Establish the task's actual scope before selecting documents. For a review
given only revisions, inspect the changed paths first.

Use this routing table as a starting point:

| paths or behavior | document |
| --- | --- |
| `openmeter/ledger/...`, credit balances, collection, correction, ledger routing | `openmeter/ledger/README.md` |
| `openmeter/billing/charges/...`, charge lifecycle, settlement, realizations | `openmeter/billing/charges/README.md` |

Use judgment about adjacent domains and how much context the task needs. For
example, if a charge change affects ledger postings, read both documents.

The table is not exhaustive; use repository structure, README links, and search
when other domain documentation may be relevant.

## Interpret docs and code together

Domain docs describe intended product semantics and architecture. Code and
tests show current behavior. Neither automatically overrides the other:

- code that contradicts a documented invariant may be the defect under review
- documentation that contradicts established behavior may be stale
- resolve the discrepancy from surrounding code, tests, history, and the
  requested product outcome before changing either

Link a specific implementation or test from the assertion it supports when
that materially shortens verification. Do not collect general navigation links
in a separate code map.

## Choose a home

- Put domain documentation in the README at the package boundary that owns the
  behavior.
- Put a narrowly scoped algorithm or implementation contract in the owning
  subpackage README when it would distract from the domain overview.
- Use `docs/` only for cross-cutting architecture or developer guidance with no
  natural package owner.
- Keep one canonical explanation. Other domains should link to it and state
  only the consequence they need.

## Write useful domain documentation

Write for an engineer who needs to understand what decisions the code is
implementing. Include only sections the domain needs:

- purpose and non-obvious vocabulary
- ownership boundaries and intentional non-ownership
- invariants and their failure consequences
- lifecycle, time, retry, or persistence semantics that affect behavior
- contracts with neighboring domains
- intentional limitations

Human-readable orientation is useful; introductory filler is not. Omit:

- package trees, method inventories, and struct field mirrors
- standalone code-entry-point sections that only catalog files or directories
- prose that merely narrates a function or declaration
- generic repository conventions and test commands
- temporary implementation state without a durable consequence
- future designs written as current architecture
- changelog narration and speculative guidance

If a fact is obvious from one declaration and carries no wider semantic
consequence, it usually belongs in code rather than the README.

## Create or revise a domain README

1. Read the existing README, current domain types, validation, lifecycle code,
   persistence mapping, representative tests, and cross-domain callers.
2. Use deleted docs and history as sources of candidate intent, never as
   current truth without verification.
3. Identify the mistake each proposed assertion prevents. Remove assertions
   with no clear consequence or evidence.
4. Rewrite around the current model instead of appending another implementation
   snapshot.
5. Reconcile overlapping docs: keep the full contract with its owner and link
   from consumers.
6. Update this skill's routing table when adding, moving, or deleting a domain
   README.
7. Review the result for stale claims, duplicated explanations, and details
   better expressed by code comments or focused developer docs.

When domain semantics change, update the relevant README in the same change.
Before deleting one, preserve any product meaning or cross-domain contract that
is not documented elsewhere.

Treat roughly 200 lines or 12 KB as a signal to edit for focus, not as a quota.
