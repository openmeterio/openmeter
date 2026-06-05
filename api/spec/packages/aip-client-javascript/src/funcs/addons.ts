import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/addons', { ...options, searchParams })
      .json<ListAddonsResponse>(),
  )
}

export function createAddon(
  client: Client,
  req: CreateAddonRequest,
  options?: RequestOptions,
): Promise<Result<CreateAddonResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/addons', { ...options, json: req })
      .json<CreateAddonResponse>(),
  )
}

export function updateAddon(
  client: Client,
  req: UpdateAddonRequest,
  options?: RequestOptions,
): Promise<Result<UpdateAddonResponse>> {
  const path = encodePath('openmeter/addons/{addonId}', {
    addonId: req.addonId,
  })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdateAddonResponse>(),
  )
}

export function getAddon(
  client: Client,
  req: GetAddonRequest,
  options?: RequestOptions,
): Promise<Result<GetAddonResponse>> {
  const path = encodePath('openmeter/addons/{addonId}', {
    addonId: req.addonId,
  })
  return request(() => http(client).get(path, options).json<GetAddonResponse>())
}

export function deleteAddon(
  client: Client,
  req: DeleteAddonRequest,
  options?: RequestOptions,
): Promise<Result<DeleteAddonResponse>> {
  const path = encodePath('openmeter/addons/{addonId}', {
    addonId: req.addonId,
  })
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function archiveAddon(
  client: Client,
  req: ArchiveAddonRequest,
  options?: RequestOptions,
): Promise<Result<ArchiveAddonResponse>> {
  const path = encodePath('openmeter/addons/{addonId}/archive', {
    addonId: req.addonId,
  })
  return request(() =>
    http(client).post(path, options).json<ArchiveAddonResponse>(),
  )
}

export function publishAddon(
  client: Client,
  req: PublishAddonRequest,
  options?: RequestOptions,
): Promise<Result<PublishAddonResponse>> {
  const path = encodePath('openmeter/addons/{addonId}/publish', {
    addonId: req.addonId,
  })
  return request(() =>
    http(client).post(path, options).json<PublishAddonResponse>(),
  )
}
