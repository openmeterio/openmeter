# Subscription Addon Diff Package

This package implements functionality to apply and restore subscription addons to subscription specifications.

## Core Components

### Diffable Interface (`diff.go`)
The central interface for objects that can be both applied to and removed from a subscription specification:
- `GetApplies()`: Returns operations that will be applied to the specification
- `GetRestores()`: Returns operations that will be applied to revert changes

### Addon Operations (`addon.go`)
Handles the conversion of subscription addons into diffable objects that can modify subscription specifications:
- `GetDiffableFromAddon()`: Creates a diffable object from a subscription addon
- `diffable`: Wrapper for apply and restore implementations for a single `SubscriptionAddonInstance`

### Application Logic (`apply.go`)
Algorithm for applying subscription addon rate cards to subscription specifications. A specification for it is as follows:

1. Given a `SubscriptionAddon`, `SubscriptionAddonInstances` can be created.
2. Given a `SubscriptionAddonInstance` and a `SubscriptionView`, the `SubscriptionAddonRateCards` can be matched to the `SubscriptionView`'s items.

Then, for each `SubscriptionAddonRateCard`:
- During the Addon's entire cadence, the RateCard has to be present: either as a modification on an existing item, or as a new item.
- The RateCard's contents should always be effective `SubscriptionAddonInstnce.Quantity` times: if quantity is 0, the RateCard should not be present.

> The merging logic of two RateCards is defined in the `subscriptionaddon` package, not here.

If the above are met for each `SubscriptionAddonRateCard` of each `SubscriptionAddonInstance`, then the `SubscriptionAddon` has been applied to the `SubscriptionSpec`.

## Usage

This package is meant to be internal to `subscription`.
