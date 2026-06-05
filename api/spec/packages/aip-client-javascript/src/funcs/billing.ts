import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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

export function listBillingProfiles(
  client: Client,
  req: ListBillingProfilesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListBillingProfilesResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
  })
  return request(() =>
    http(client)
      .get('openmeter/profiles', { ...options, searchParams })
      .json<ListBillingProfilesResponse>(),
  )
}

export function createBillingProfile(
  client: Client,
  req: CreateBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<CreateBillingProfileResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/profiles', { ...options, json: req })
      .json<CreateBillingProfileResponse>(),
  )
}

export function getBillingProfile(
  client: Client,
  req: GetBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<GetBillingProfileResponse>> {
  const path = encodePath('openmeter/profiles/{id}', { id: req.id })
  return request(() =>
    http(client)
      .get(path, options)
      .json<GetBillingProfileResponse>(),
  )
}

export function updateBillingProfile(
  client: Client,
  req: UpdateBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<UpdateBillingProfileResponse>> {
  const path = encodePath('openmeter/profiles/{id}', { id: req.id })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdateBillingProfileResponse>(),
  )
}

export function deleteBillingProfile(
  client: Client,
  req: DeleteBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<DeleteBillingProfileResponse>> {
  const path = encodePath('openmeter/profiles/{id}', { id: req.id })
  return request(async () => {
    await http(client).delete(path, options)
  })
}
