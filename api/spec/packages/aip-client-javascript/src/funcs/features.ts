import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListFeaturesRequest,
  ListFeaturesResponse,
  CreateFeatureRequest,
  CreateFeatureResponse,
  GetFeatureRequest,
  GetFeatureResponse,
  UpdateFeatureRequest,
  UpdateFeatureResponse,
  DeleteFeatureRequest,
  DeleteFeatureResponse,
  QueryFeatureCostRequest,
  QueryFeatureCostResponse,
} from '../models/operations/features.js'

export function listFeatures(
  client: Client,
  req: ListFeaturesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListFeaturesResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/features', { ...options, searchParams })
      .json<ListFeaturesResponse>(),
  )
}

export function createFeature(
  client: Client,
  req: CreateFeatureRequest,
  options?: RequestOptions,
): Promise<Result<CreateFeatureResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/features', { ...options, json: req })
      .json<CreateFeatureResponse>(),
  )
}

export function getFeature(
  client: Client,
  req: GetFeatureRequest,
  options?: RequestOptions,
): Promise<Result<GetFeatureResponse>> {
  const path = encodePath('openmeter/features/{featureId}', {
    featureId: req.featureId,
  })
  return request(() =>
    http(client).get(path, options).json<GetFeatureResponse>(),
  )
}

export function updateFeature(
  client: Client,
  req: UpdateFeatureRequest,
  options?: RequestOptions,
): Promise<Result<UpdateFeatureResponse>> {
  const path = encodePath('openmeter/features/{featureId}', {
    featureId: req.featureId,
  })
  return request(() =>
    http(client)
      .patch(path, { ...options, json: req.body })
      .json<UpdateFeatureResponse>(),
  )
}

export function deleteFeature(
  client: Client,
  req: DeleteFeatureRequest,
  options?: RequestOptions,
): Promise<Result<DeleteFeatureResponse>> {
  const path = encodePath('openmeter/features/{featureId}', {
    featureId: req.featureId,
  })
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function queryFeatureCost(
  client: Client,
  req: QueryFeatureCostRequest,
  options?: RequestOptions,
): Promise<Result<QueryFeatureCostResponse>> {
  const path = encodePath('openmeter/features/{featureId}/cost/query', {
    featureId: req.featureId,
  })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<QueryFeatureCostResponse>(),
  )
}
