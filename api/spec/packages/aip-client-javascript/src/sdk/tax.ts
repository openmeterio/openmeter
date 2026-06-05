import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  createTaxCode,
  getTaxCode,
  listTaxCodes,
  upsertTaxCode,
  deleteTaxCode,
} from '../funcs/tax.js'
import type {
  CreateTaxCodeRequest,
  CreateTaxCodeResponse,
  GetTaxCodeRequest,
  GetTaxCodeResponse,
  ListTaxCodesRequest,
  ListTaxCodesResponse,
  UpsertTaxCodeRequest,
  UpsertTaxCodeResponse,
  DeleteTaxCodeRequest,
  DeleteTaxCodeResponse,
} from '../models/operations/tax.js'

export class Tax {
  constructor(private readonly _client: Client) {}

  async createCode(
    request: CreateTaxCodeRequest,
    options?: RequestOptions,
  ): Promise<CreateTaxCodeResponse> {
    return unwrap(await createTaxCode(this._client, request, options))
  }

  async getCode(
    request: GetTaxCodeRequest,
    options?: RequestOptions,
  ): Promise<GetTaxCodeResponse> {
    return unwrap(await getTaxCode(this._client, request, options))
  }

  async listCodes(
    request?: ListTaxCodesRequest,
    options?: RequestOptions,
  ): Promise<ListTaxCodesResponse> {
    return unwrap(await listTaxCodes(this._client, request, options))
  }

  async upsertCode(
    request: UpsertTaxCodeRequest,
    options?: RequestOptions,
  ): Promise<UpsertTaxCodeResponse> {
    return unwrap(await upsertTaxCode(this._client, request, options))
  }

  async deleteCode(
    request: DeleteTaxCodeRequest,
    options?: RequestOptions,
  ): Promise<DeleteTaxCodeResponse> {
    return unwrap(await deleteTaxCode(this._client, request, options))
  }
}
