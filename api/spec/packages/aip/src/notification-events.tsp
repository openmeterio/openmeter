import "@openmeter/typespec-notification-events";
// Common aliases the top-level Billing namespace and Shared depends on Common,
// so every compilation touching Shared needs Billing in the program.
import "./common/error.tsp";
import "./common/pagination.tsp";
import "./billing/index.tsp";
import "./billing/notification-events/index.tsp";
import "./entitlements/notification-events/index.tsp";

using NotificationCodegen;

// Maps every Notification Event namespace to the Go package generated from it.
// Adding a producing domain means importing its notification-events folder and
// adding one @@goPackage here; scripts/generate-notification-events.mjs picks
// it up.
@@goPackage(
  Entitlements.NotificationEvents,
  "github.com/openmeterio/openmeter/openmeter/entitlement/notificationevents"
);

@@goPackage(
  Billing.NotificationEvents,
  "github.com/openmeterio/openmeter/openmeter/billing/notificationevents"
);
