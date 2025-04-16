namespace OpenMeter.App;

/**
 * Custom Invoicing app can be used for interface with any invoicing or payment system.
 *
 * This app provides ways to manipulate invoices and payments, however the integration
 * must rely on Notifications API to get notified about invoice changes.
 */
@friendlyName("CustomInvoicingApp")
model CustomInvoicingApp {
  ...AppBase;

  /**
   * The app's type is CustomInvoicing.
   */
  type: Type.CustomInvoicing;

  /**
   * Skip draft.sync hook.
   *
   * The hook will auto progress to the next state of the invoice.
   */
  skipDraftSyncHook: boolean;

  /**
   * Skip issuing.sync hook.
   *
   * The hook will auto progress to the next state of the invoice.
   */
  skipIssuingSyncHook: boolean;
}
