import "../shared/index.tsp";
import "./app.tsp";
import "../invoices/index.tsp";

namespace Apps;

/**
 * External Invoicing app enables integration with third-party invoicing or payment system.
 *
 * The app supports a bi-directional synchronization pattern where the billing system manages
 * the invoice lifecycle while the external system handles invoice presentation and payment collection.
 *
 * Integration workflow:
 * 1. The billing system creates invoices and transitions them through lifecycle states (draft → issuing → issued)
 * 2. The integration receives webhook notifications about invoice state changes
 * 3. The integration calls back to provide external system IDs and metadata
 * 4. The integration reports payment events back via the payment status API
 *
 * State synchronization is controlled by hooks that pause invoice progression until the
 * external system confirms synchronization via API callbacks.
 */
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "Hooks are conceptually separate settings"
@friendlyName("BillingAppExternalInvoicing")
model AppExternalInvoicing {
  ...AppBase<AppType.ExternalInvoicing>;

  /**
   * Enable draft synchronization hook.
   *
   * When enabled, invoices will pause at the draft state and wait for the integration
   * to call the draft synchronized endpoint before progressing to the issuing state.
   * This allows the external system to validate and prepare the invoice data.
   *
   * When disabled, invoices automatically progress through the draft state based on
   * the configured workflow timing.
   */
  enable_draft_sync_hook: boolean;

  /**
   * Enable issuing synchronization hook.
   *
   * When enabled, invoices will pause at the issuing state and wait for the integration
   * to call the issuing synchronized endpoint before progressing to the issued state.
   * This ensures the external invoicing system has successfully created and finalized
   * the invoice before it is marked as issued.
   *
   * When disabled, invoices automatically progress through the issuing state and are
   * immediately marked as issued.
   */
  enable_issuing_sync_hook: boolean;
}
