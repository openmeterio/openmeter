# Subject-Customer Consolidation

OpenMeter has always separated usage owners (subjects) from billing
entities (customers). This design stays, but now you can assign multiple
subjects to a customer.

To reduce complexity, usage sent with a Customer ID or Key is now
automatically mapped to that customer.

We are also unifying subjects and customers for entitlements and
notifications, making customer APIs the primary interface.

## 🔄 What's Changing

### 🔗 1. Consolidating Subject and Customer Entities

Customers now replace subjects as the primary entity. Mapping subjects
to customers is optional and does not affect the ingest API or the
`subject` field in usage events.

If the subject is a Customer ID or Key, OpenMeter automatically
attributes usage to that customer. Custom subjects can still be created
and linked to customers when needed.

→ Use customer entities instead of subjects.

> [!NOTE]
> Ensure your custom subject values do not collide with any existing `customer.key`
> or `customer.id`. If a collision occurs, usage may be attributed to the wrong customer.

#### Before

Mandatory subject-customer mapping via `usageAttribution` property before the change.

```ts
const customer = await openmeter.customers.create({
  name: 'ACME, Inc.',
  key: 'my-identifier',
  usageAttribution: { subjects: ['my-identifier'] },
});

await openmeter.events.ingest({
  type: 'prompt',
  // customer.usageAttribution.subjects
  subject: 'my-identifier',
  data: { tokens: 10 },
});
```

#### After

Optional definition of `usageAttribution` property on customer and
automatic usage attribution when usage event uses Customer ID or Key.

```ts
const customer = await openmeter.customers.create({
  name: 'ACME, Inc.',
  key: 'my-identifier',
  // optional: usageAttribution
});

await openmeter.events.ingest({
  type: 'prompt',
  // customer.id, customer.key, or usageAttribution.subjects
  subject: 'my-identifier',
  data: { tokens: 10 },
});
```

### ➡️ 2. Deprecating Subject Entity APIs

All endpoints starting with `/api/v*/subjects...` will be deprecated
after the migration period.

→ Use customer APIs instead: entitlements and notification

#### Before (subject-level)

```ts
const entitlement = await openmeter.subjects.createEntitlement('my-identifier', { … });
```

#### After (customer-level)

```ts
const entitlement = await openmeter.customers.createEntitlement('my-identifier', { … });
```

### ✏️ 3. Defining subjects on a customer becomes optional

The `usageAttribution` field on customers is now optional to set.

→ OpenMeter will automatically attribute usage by customer ID or Key.

### 👥 4. Assigning multiple subjects to a customer

You can now assign multiple subjects to a single customer. This is
useful when multiple consumers need to be billed together.

```ts
const customer = await openmeter.customers.create({
  name: 'ACME, Inc.',
  key: 'my-identifier',
  usageAttribution: {
    subjects: [
      'my-subject-1',
      'my-subject-2',
      'my-subject-3'
    ]
  },
});
```

## 🗓 Deprecation Timeline

| Date               | Change                                                         |
|--------------------|----------------------------------------------------------------|
| September 01, 2025 | Subject APIs marked as deprecated, functionality unchanged     |
| November 01, 2025  | Subject APIs removed entirely                                  |

## Migration Guide

Who should act? Anyone calling /subjects APIs or relying on subject-level entitlements/notifications.

- **1. Switch to customer APIs**
  - Find all calls to `/api/v*/subjects*.`
  - Entitlements → Use customer APIs
  - Notifications → Use customer field in events
- **2. Decide your attribution strategy**
  - Easiest: Start sending `subject = customer.key` (`or customer.id`) in new events.
  - Custom subjects: Keep your own subject strings and add them to `customer.usageAttribution.subjects`.

## Frequently Asked Questions

### What is a subject?

A subject is any entity that generates metered usage in OpenMeter — for
example, a customer, user, server, service, or device. Subjects exist
only in the context of usage events and metering, representing the
consumers of your service.

### What is a customer?

A customer is the individual or organization that uses your service and
is billed for the usage. One customer can have one or more subjects
assigned to it to track usage for these consumers and bill them together.

### Will my existing integrations break after the migration?

No. Existing subject-based APIs will continue to work until the
final deprecation date. However, we recommend updating to customer APIs
as soon as possible.

### Will you create customers for my subjects automatically?

Yes. All subjects not assigned to customers will be turned into
customers on OpenMeter Cloud. These customers remain in sync with
the subjects they were created from.

### Can I assign multiple subjects to the same customer?

Yes, this is now supported natively. You can assign multiple subjects to a
single customer via the `usageAttribution` API property.

### Can I assign the same subject to multiple customers?

No, like before one subject can only belong to a single customer.

### Can I still assign subjects to customers later?

Yes. You can still ingest usage with unassigned subjects, and later link
them to customers. You can now assign multiple subjects to customers too.

### Can I mix customer IDs and subject IDs in ingestion?

Yes. You can use Customer ID, Customer Key or manually assigned
subjects as the `subject` field in the usage event.

### Can I have customer-level entitlements?

Yes. This change introduces customer-level entitlements, which combine
usage from all subjects assigned to a customer when calculating
balances.

### Can I still create subject-level entitlements?

For now, yes. But subject level entitlements APIs will eventually be deprecated.
In the future, use customer-level entitlements, with single or multiple subjects as needed.

### Can I revert back to using subjects APIs only?

No. After the final removal date, the subject APIs
like subject-level entitlements will no longer be available.
