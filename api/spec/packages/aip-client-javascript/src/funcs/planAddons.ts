import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListPlanAddonsRequest,
  ListPlanAddonsResponse,
  CreatePlanAddonRequest,
  CreatePlanAddonResponse,
  GetPlanAddonRequest,
  GetPlanAddonResponse,
  UpdatePlanAddonRequest,
  UpdatePlanAddonResponse,
  DeletePlanAddonRequest,
  DeletePlanAddonResponse,
} from '../models/operations/planAddons.js'

export function listPlanAddons(
  client: Client,
  req: ListPlanAddonsRequest,
  options?: RequestOptions,
): Promise<Result<ListPlanAddonsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
  })
  const path = encodePath('openmeter/plans/{planId}/addons', { planId: req.planId })
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<ListPlanAddonsResponse>(),
  )
}

export function createPlanAddon(
  client: Client,
  req: CreatePlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<CreatePlanAddonResponse>> {
  const path = encodePath('openmeter/plans/{planId}/addons', { planId: req.planId })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreatePlanAddonResponse>(),
  )
}

export function getPlanAddon(
  client: Client,
  req: GetPlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<GetPlanAddonResponse>> {
  const path = encodePath('openmeter/plans/{planId}/addons/{planAddonId}', { planId: req.planId, planAddonId: req.planAddonId })
  return request(() =>
    http(client)
      .get(path, options)
      .json<GetPlanAddonResponse>(),
  )
}

export function updatePlanAddon(
  client: Client,
  req: UpdatePlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<UpdatePlanAddonResponse>> {
  const path = encodePath('openmeter/plans/{planId}/addons/{planAddonId}', { planId: req.planId, planAddonId: req.planAddonId })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdatePlanAddonResponse>(),
  )
}

export function deletePlanAddon(
  client: Client,
  req: DeletePlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<DeletePlanAddonResponse>> {
  const path = encodePath('openmeter/plans/{planId}/addons/{planAddonId}', { planId: req.planId, planAddonId: req.planAddonId })
  return request(async () => {
    await http(client).delete(path, options)
  })
}
