import { z } from 'zod'
import * as schemas from '../schemas.js'

export type GetInvoiceRequest = {
  invoiceId: string
}
export type GetInvoiceResponse = z.output<typeof schemas.getInvoiceResponse>
