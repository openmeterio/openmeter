import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  return request(() => {
    const query = toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listFeaturesQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listFeaturesQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get('openmeter/features', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listFeaturesResponseWire, data)
        }
        return fromWire(data, schemas.listFeaturesResponse)
      })
  })
}

export function createFeature(
  client: Client,
  req: CreateFeatureRequest,
  options?: RequestOptions,
): Promise<Result<CreateFeatureResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createFeatureBody)
    if (client._options.validate) {
      assertValid(schemas.createFeatureBodyWire, body)
    }
    return http(client)
      .post('openmeter/features', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createFeatureResponseWire, data)
        }
        return fromWire(data, schemas.createFeatureResponse)
      })
  })
}

export function getFeature(
  client: Client,
  req: GetFeatureRequest,
  options?: RequestOptions,
): Promise<Result<GetFeatureResponse>> {
  const path = `openmeter/features/${(() => {
    if (req.featureId === undefined) {
      throw new Error('missing path parameter: featureId')
    }
    return encodeURIComponent(String(req.featureId))
  })()}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getFeatureResponseWire, data)
        }
        return fromWire(data, schemas.getFeatureResponse)
      }),
  )
}

export function updateFeature(
  client: Client,
  req: UpdateFeatureRequest,
  options?: RequestOptions,
): Promise<Result<UpdateFeatureResponse>> {
  const path = `openmeter/features/${(() => {
    if (req.featureId === undefined) {
      throw new Error('missing path parameter: featureId')
    }
    return encodeURIComponent(String(req.featureId))
  })()}`
  return request(() => {
    const body = toWire(req.body, schemas.updateFeatureBody)
    if (client._options.validate) {
      assertValid(schemas.updateFeatureBodyWire, body)
    }
    return http(client)
      .patch(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateFeatureResponseWire, data)
        }
        return fromWire(data, schemas.updateFeatureResponse)
      })
  })
}

export function deleteFeature(
  client: Client,
  req: DeleteFeatureRequest,
  options?: RequestOptions,
): Promise<Result<DeleteFeatureResponse>> {
  const path = `openmeter/features/${(() => {
    if (req.featureId === undefined) {
      throw new Error('missing path parameter: featureId')
    }
    return encodeURIComponent(String(req.featureId))
  })()}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function queryFeatureCost(
  client: Client,
  req: QueryFeatureCostRequest,
  options?: RequestOptions,
): Promise<Result<QueryFeatureCostResponse>> {
  const path = `openmeter/features/${(() => {
    if (req.featureId === undefined) {
      throw new Error('missing path parameter: featureId')
    }
    return encodeURIComponent(String(req.featureId))
  })()}/cost/query`
  return request(() => {
    const body = toWire(req.body, schemas.queryFeatureCostBody)
    if (client._options.validate) {
      assertValid(schemas.queryFeatureCostBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.queryFeatureCostResponseWire, data)
        }
        return fromWire(data, schemas.queryFeatureCostResponse)
      })
  })
}
