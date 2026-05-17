namespace Shared;

const MeteringAndBillingTitle = "Metering & Billing";
const OpenMeterTitle = "OpenMeter";

const MetersTag = "Meters";
const MetersDescription = "Meters specify how to aggregate events for billing and analytics purposes. Meters can be configured with multiple aggregation methods and groupings. Multiple meters can be created for the same event type, enabling flexible metering scenarios.";

const EventsTag = "Metering Events";
const EventsDescription = "Metering events are used to track usage of your product or service. Events are processed asynchronously by the meters, so they may not be immediately available for querying.";

const CustomersTag = "OpenMeter Customers";
const CustomersDescription = "Customers are used to track usage of your product or service. Customers can be individuals or organizations that can subscribe to plans and have access to features.";

const EntitlementsTag = "OpenMeter Entitlements";
const EntitlementsDescription = "Entitlements are used to control access to features for customers.";

const SubscriptionsTag = "OpenMeter Subscriptions";
const SubscriptionsDescription = "Subscriptions are used to track usage of your product or service. Subscriptions can be individuals or organizations that can subscribe to plans and have access to features.";

const AppsTag = "OpenMeter Apps";
const AppsDescription = "Apps enable you to extend and customize billing and usage workflows by integrating with external systems and services. Apps can automate and enhance your billing ecosystem by supporting capabilities such as synchronizing usage data with third-party platforms, calculating taxes, generating and delivering invoices, handling payment collection, and other billing-related tasks.";

const BillingTag = "OpenMeter Billing";
const BillingDescription = "Billing manages the billing profiles, currencies, cost bases, and invoices for customers.";

const TaxTag = "OpenMeter Tax";
const TaxDescription = "Tax codes are used to calculate taxes for customers.";

const LLMCostTag = "OpenMeter LLM Cost";
const LLMCostDescription = "LLM cost database providing normalized pricing data for language models across providers. Prices are synced from multiple external sources and auto-confirmed when they agree.";

const FeaturesTag = "OpenMeter Features";
const FeaturesDescription = "Features represent product capabilities backed by meters, with optional per-unit cost configuration.";

const ProductCatalogTag = "OpenMeter Product Catalog";
const ProductCatalogDescription = "Product catalog manages plans, add-ons, and their associations for subscription-based billing.";

const DefaultsTag = "OpenMeter Defaults";
const DefaultsDescription = "Organization-level default configuration.";

const GovernanceTag = "OpenMeter Governance";
const GovernanceDescription = "Governance evaluation of customers to check their feature access.";

const AccountingTag = "OpenMeter Accounting";
const AccountingDescription = "Accounting manages the accounting periods, transactions, and revenue recognition.";

// The functionality should be considered unstable.
const UnstableExtension = "x-unstable";
// The functionality is internal and should not be used by customers.
const InternalExtension = "x-internal";
// The functionality is private and should not be exposed to customers.
const PrivateExtension = "x-private";
