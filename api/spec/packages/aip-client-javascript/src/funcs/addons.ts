import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  ListAddonsRequest,
  ListAddonsResponse,
  CreateAddonRequest,
  CreateAddonResponse,
  UpdateAddonRequest,
  UpdateAddonResponse,
  GetAddonRequest,
  GetAddonResponse,
  DeleteAddonRequest,
  DeleteAddonResponse,
  ArchiveAddonRequest,
  ArchiveAddonResponse,
  PublishAddonRequest,
  PublishAddonResponse,
} from '../models/operations/addons.js'

export function listAddons(
  client: Client,
  req: ListAddonsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListAddonsResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listAddonsQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/addons', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listAddonsResponseWire, data)
        }
        return fromWire(data, schemas.listAddonsResponse)
      }),
  )
}

export function createAddon(
  client: Client,
  req: CreateAddonRequest,
  options?: RequestOptions,
): Promise<Result<CreateAddonResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createAddonBody)
    if (client._options.validate) {
      assertValid(schemas.createAddonBodyWire, body)
    }
    return http(client)
      .post('openmeter/addons', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createAddonResponseWire, data)
        }
        return fromWire(data, schemas.createAddonResponse)
      })
  })
}

export function updateAddon(
  client: Client,
  req: UpdateAddonRequest,
  options?: RequestOptions,
): Promise<Result<UpdateAddonResponse>> {
  const path = `openmeter/addons/${encodeURIComponent(String(req.addonId))}`
  return request(() => {
    const body = toWire(req.body, schemas.updateAddonBody)
    if (client._options.validate) {
      assertValid(schemas.updateAddonBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateAddonResponseWire, data)
        }
        return fromWire(data, schemas.updateAddonResponse)
      })
  })
}

export function getAddon(
  client: Client,
  req: GetAddonRequest,
  options?: RequestOptions,
): Promise<Result<GetAddonResponse>> {
  const path = `openmeter/addons/${encodeURIComponent(String(req.addonId))}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getAddonResponseWire, data)
        }
        return fromWire(data, schemas.getAddonResponse)
      }),
  )
}

export function deleteAddon(
  client: Client,
  req: DeleteAddonRequest,
  options?: RequestOptions,
): Promise<Result<DeleteAddonResponse>> {
  const path = `openmeter/addons/${encodeURIComponent(String(req.addonId))}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function archiveAddon(
  client: Client,
  req: ArchiveAddonRequest,
  options?: RequestOptions,
): Promise<Result<ArchiveAddonResponse>> {
  const path = `openmeter/addons/${encodeURIComponent(String(req.addonId))}/archive`
  return request(() =>
    http(client)
      .post(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.archiveAddonResponseWire, data)
        }
        return fromWire(data, schemas.archiveAddonResponse)
      }),
  )
}

export function publishAddon(
  client: Client,
  req: PublishAddonRequest,
  options?: RequestOptions,
): Promise<Result<PublishAddonResponse>> {
  const path = `openmeter/addons/${encodeURIComponent(String(req.addonId))}/publish`
  return request(() =>
    http(client)
      .post(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.publishAddonResponseWire, data)
        }
        return fromWire(data, schemas.publishAddonResponse)
      }),
  )
}
