import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listFeatures,
  createFeature,
  getFeature,
  updateFeature,
  deleteFeature,
  queryFeatureCost,
} from '../funcs/features.js'
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

export class Features {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListFeaturesRequest,
    options?: RequestOptions,
  ): Promise<ListFeaturesResponse> {
    return unwrap(await listFeatures(this._client, request, options))
  }

  async create(
    request: CreateFeatureRequest,
    options?: RequestOptions,
  ): Promise<CreateFeatureResponse> {
    return unwrap(await createFeature(this._client, request, options))
  }

  async get(
    request: GetFeatureRequest,
    options?: RequestOptions,
  ): Promise<GetFeatureResponse> {
    return unwrap(await getFeature(this._client, request, options))
  }

  async update(
    request: UpdateFeatureRequest,
    options?: RequestOptions,
  ): Promise<UpdateFeatureResponse> {
    return unwrap(await updateFeature(this._client, request, options))
  }

  async delete(
    request: DeleteFeatureRequest,
    options?: RequestOptions,
  ): Promise<DeleteFeatureResponse> {
    return unwrap(await deleteFeature(this._client, request, options))
  }

  async queryCost(
    request: QueryFeatureCostRequest,
    options?: RequestOptions,
  ): Promise<QueryFeatureCostResponse> {
    return unwrap(await queryFeatureCost(this._client, request, options))
  }
}
