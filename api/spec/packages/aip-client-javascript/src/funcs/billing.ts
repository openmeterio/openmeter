import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
      },
      schemas.listBillingProfilesQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/profiles', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listBillingProfilesResponseWire, data)
        }
        return fromWire(data, schemas.listBillingProfilesResponse)
      }),
  )
}

export function createBillingProfile(
  client: Client,
  req: CreateBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<CreateBillingProfileResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createBillingProfileBody)
    if (client._options.validate) {
      assertValid(schemas.createBillingProfileBodyWire, body)
    }
    return http(client)
      .post('openmeter/profiles', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createBillingProfileResponseWire, data)
        }
        return fromWire(data, schemas.createBillingProfileResponse)
      })
  })
}

export function getBillingProfile(
  client: Client,
  req: GetBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<GetBillingProfileResponse>> {
  const path = `openmeter/profiles/${encodeURIComponent(String(req.id))}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getBillingProfileResponseWire, data)
        }
        return fromWire(data, schemas.getBillingProfileResponse)
      }),
  )
}

export function updateBillingProfile(
  client: Client,
  req: UpdateBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<UpdateBillingProfileResponse>> {
  const path = `openmeter/profiles/${encodeURIComponent(String(req.id))}`
  return request(() => {
    const body = toWire(req.body, schemas.updateBillingProfileBody)
    if (client._options.validate) {
      assertValid(schemas.updateBillingProfileBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateBillingProfileResponseWire, data)
        }
        return fromWire(data, schemas.updateBillingProfileResponse)
      })
  })
}

export function deleteBillingProfile(
  client: Client,
  req: DeleteBillingProfileRequest,
  options?: RequestOptions,
): Promise<Result<DeleteBillingProfileResponse>> {
  const path = `openmeter/profiles/${encodeURIComponent(String(req.id))}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}
