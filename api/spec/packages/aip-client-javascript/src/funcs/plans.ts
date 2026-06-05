import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListPlansRequest,
  ListPlansResponse,
  CreatePlanRequest,
  CreatePlanResponse,
  UpdatePlanRequest,
  UpdatePlanResponse,
  GetPlanRequest,
  GetPlanResponse,
  DeletePlanRequest,
  DeletePlanResponse,
  ArchivePlanRequest,
  ArchivePlanResponse,
  PublishPlanRequest,
  PublishPlanResponse,
} from '../models/operations/plans.js'

export function listPlans(
  client: Client,
  req: ListPlansRequest = {},
  options?: RequestOptions,
): Promise<Result<ListPlansResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/plans', { ...options, searchParams })
      .json<ListPlansResponse>(),
  )
}

export function createPlan(
  client: Client,
  req: CreatePlanRequest,
  options?: RequestOptions,
): Promise<Result<CreatePlanResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/plans', { ...options, json: req })
      .json<CreatePlanResponse>(),
  )
}

export function updatePlan(
  client: Client,
  req: UpdatePlanRequest,
  options?: RequestOptions,
): Promise<Result<UpdatePlanResponse>> {
  const path = encodePath('openmeter/plans/{planId}', { planId: req.planId })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdatePlanResponse>(),
  )
}

export function getPlan(
  client: Client,
  req: GetPlanRequest,
  options?: RequestOptions,
): Promise<Result<GetPlanResponse>> {
  const path = encodePath('openmeter/plans/{planId}', { planId: req.planId })
  return request(() => http(client).get(path, options).json<GetPlanResponse>())
}

export function deletePlan(
  client: Client,
  req: DeletePlanRequest,
  options?: RequestOptions,
): Promise<Result<DeletePlanResponse>> {
  const path = encodePath('openmeter/plans/{planId}', { planId: req.planId })
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function archivePlan(
  client: Client,
  req: ArchivePlanRequest,
  options?: RequestOptions,
): Promise<Result<ArchivePlanResponse>> {
  const path = encodePath('openmeter/plans/{planId}/archive', {
    planId: req.planId,
  })
  return request(() =>
    http(client).post(path, options).json<ArchivePlanResponse>(),
  )
}

export function publishPlan(
  client: Client,
  req: PublishPlanRequest,
  options?: RequestOptions,
): Promise<Result<PublishPlanResponse>> {
  const path = encodePath('openmeter/plans/{planId}/publish', {
    planId: req.planId,
  })
  return request(() =>
    http(client).post(path, options).json<PublishPlanResponse>(),
  )
}
