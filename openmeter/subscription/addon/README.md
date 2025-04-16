# Subscription Addon

This package contains the subscription addon related functionality.

## Entity Relationship Diagram

```mermaid
erDiagram
    Subscription ||--|| SubscriptionAddon : "has (1:1)"
    SubscriptionAddon ||--o{ SubscriptionAddonQuantity : "has (1:N)"
    SubscriptionAddon ||--o{ SubscriptionAddonRateCard : "has (1:N) calculated from Addon"
    Addon ||--|| SubscriptionAddon : "has (1:1)"

    Subscription {
        string id PK
    }

    Addon {
        string id PK
    }

    SubscriptionAddon {
        string id PK
        string subscription_id FK
        string addon_id FK
    }

    SubscriptionAddonQuantity {
        string id PK
        string subscription_addon_id FK
    }

    SubscriptionAddonRateCard {

    }
```

## Quirks

1. **Feature resolution**: When an addon creates a new SubscriptionItem (not a split of an existing one but a new item), the featureKey => feature resolution will happen at sync time. This means, that potentially, in a subscription, items with the same featureKey reference can point to different feature instances.