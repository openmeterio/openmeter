import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listInvoices,
  getInvoice,
  updateInvoice,
  deleteInvoice,
  advanceInvoice,
  approveInvoice,
  retryInvoice,
  snapshotQuantitiesInvoice,
} from '../funcs/invoices.js'
import type {
  ListInvoicesRequest,
  ListInvoicesResponse,
  GetInvoiceRequest,
  GetInvoiceResponse,
  UpdateInvoiceRequest,
  UpdateInvoiceResponse,
  DeleteInvoiceRequest,
  DeleteInvoiceResponse,
  AdvanceInvoiceRequest,
  AdvanceInvoiceResponse,
  ApproveInvoiceRequest,
  ApproveInvoiceResponse,
  RetryInvoiceRequest,
  RetryInvoiceResponse,
  SnapshotQuantitiesInvoiceRequest,
  SnapshotQuantitiesInvoiceResponse,
} from '../models/operations/invoices.js'

export class Invoices {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListInvoicesRequest,
    options?: RequestOptions,
  ): Promise<ListInvoicesResponse> {
    return unwrap(await listInvoices(this._client, request, options))
  }

  async get(
    request: GetInvoiceRequest,
    options?: RequestOptions,
  ): Promise<GetInvoiceResponse> {
    return unwrap(await getInvoice(this._client, request, options))
  }

  async update(
    request: UpdateInvoiceRequest,
    options?: RequestOptions,
  ): Promise<UpdateInvoiceResponse> {
    return unwrap(await updateInvoice(this._client, request, options))
  }

  async delete(
    request: DeleteInvoiceRequest,
    options?: RequestOptions,
  ): Promise<DeleteInvoiceResponse> {
    return unwrap(await deleteInvoice(this._client, request, options))
  }

  async advance(
    request: AdvanceInvoiceRequest,
    options?: RequestOptions,
  ): Promise<AdvanceInvoiceResponse> {
    return unwrap(await advanceInvoice(this._client, request, options))
  }

  async approve(
    request: ApproveInvoiceRequest,
    options?: RequestOptions,
  ): Promise<ApproveInvoiceResponse> {
    return unwrap(await approveInvoice(this._client, request, options))
  }

  async retry(
    request: RetryInvoiceRequest,
    options?: RequestOptions,
  ): Promise<RetryInvoiceResponse> {
    return unwrap(await retryInvoice(this._client, request, options))
  }

  async snapshotQuantities(
    request: SnapshotQuantitiesInvoiceRequest,
    options?: RequestOptions,
  ): Promise<SnapshotQuantitiesInvoiceResponse> {
    return unwrap(
      await snapshotQuantitiesInvoice(this._client, request, options),
    )
  }
}
