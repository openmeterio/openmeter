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
   * Enable draft.sync hook.
   *
   * If the hook is not enabled, the invoice will be progressed to the next state automatically.
   */
  enableDraftSyncHook: boolean;

  /**
   * Enable issuing.sync hook.
   *
   * If the hook is not enabled, the invoice will be progressed to the next state automatically.
   */
  enableIssuingSyncHook: boolean;
}
