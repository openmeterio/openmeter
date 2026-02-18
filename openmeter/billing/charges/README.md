# Realizations

Realizations are the basic types of fulfilling any obligation represented by the charge. Types of realizations:

| Type | Description | Example |
+----+-----------+-------+
| CreditRecognition | Fulfilled by consuming the customer's credit balance, there's no settlement needed, as the system settles the possible credit transfers | |
| InvoiceLaterRecognition | Represents an amount charge that will be eventually paid | The customer have used $10 in the service period of [2026-01-01..2026-01-02] that we are going to invoice the customer for once the billing period happens |
| StandardInvoiceSettlement | Represents a request for the end user that will be eventually payed using a standard invoice | The customer has $20 worth of InvoiceLaterRealization so we send him an invoice with that amount |
| ExternalSettlement | Represents a settlement that is performed on the external system | e.g. the user's own payment gateway handles the settlement and we are just notified of it's status |

# Usage Based charges

## Revenue recognition train

Usage based charges are periodically monitored for incoming events.

Each day a RecognizeRevenue call is issued to recognize any outstanding revenue for that period.

The revenue can be recognized in two ways:
- CreditRealization
- StandardInvoiceObligationRealization

## Creating an invoice

Note: this is only valid for credit_then_invoice mode.

Once an invoice should be created (e.g. InvoicePendingLines, we are at the invoice_at of a charge), the following logic is executed:
- The service period for the invoice is determined based on the [max(charge.ServicePeriod.From, lastInvoice.ServicePeriod.To)...AsOf]
- A revenue recognition is performed up to the AsOf of the Invoice Pending lines
- (optionally): All StandardInvoiceObligationRealizations are tried to fulfilled using credits
- An invoice is created creating the StandardInvoiceRealization entry for the charge
