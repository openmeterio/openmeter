import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listBillingProfiles,
  createBillingProfile,
  getBillingProfile,
  updateBillingProfile,
  deleteBillingProfile,
} from '../funcs/billing.js'
import type {
  ListBillingProfilesRequest,
  ListBillingProfilesResponse,
  CreateBillingProfileRequest,
  CreateBillingProfileResponse,
  GetBillingProfileRequest,
  GetBillingProfileResponse,
  UpdateBillingProfileRequest,
  UpdateBillingProfileResponse,
  DeleteBillingProfileRequest,
  DeleteBillingProfileResponse,
} from '../models/operations/billing.js'

export class Billing {
  constructor(private readonly _client: Client) {}

  async listProfiles(
    request?: ListBillingProfilesRequest,
    options?: RequestOptions,
  ): Promise<ListBillingProfilesResponse> {
    return unwrap(await listBillingProfiles(this._client, request, options))
  }

  async createProfile(
    request: CreateBillingProfileRequest,
    options?: RequestOptions,
  ): Promise<CreateBillingProfileResponse> {
    return unwrap(await createBillingProfile(this._client, request, options))
  }

  async getProfile(
    request: GetBillingProfileRequest,
    options?: RequestOptions,
  ): Promise<GetBillingProfileResponse> {
    return unwrap(await getBillingProfile(this._client, request, options))
  }

  async updateProfile(
    request: UpdateBillingProfileRequest,
    options?: RequestOptions,
  ): Promise<UpdateBillingProfileResponse> {
    return unwrap(await updateBillingProfile(this._client, request, options))
  }

  async deleteProfile(
    request: DeleteBillingProfileRequest,
    options?: RequestOptions,
  ): Promise<DeleteBillingProfileResponse> {
    return unwrap(await deleteBillingProfile(this._client, request, options))
  }
}
