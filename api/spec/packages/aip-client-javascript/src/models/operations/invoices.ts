import { z } from 'zod'
import * as schemas from '../schemas.js'

export type GetBillingInvoiceRequest = {
  invoiceId: string
}
export type GetBillingInvoiceResponse = z.output<
  typeof schemas.getBillingInvoiceResponse
>
