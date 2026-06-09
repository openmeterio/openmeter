import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  createMeter,
  getMeter,
  listMeters,
  updateMeter,
  deleteMeter,
  queryMeter,
} from '../funcs/meters.js'
import type {
  CreateMeterRequest,
  CreateMeterResponse,
  GetMeterRequest,
  GetMeterResponse,
  ListMetersRequest,
  ListMetersResponse,
  UpdateMeterRequest,
  UpdateMeterResponse,
  DeleteMeterRequest,
  DeleteMeterResponse,
  QueryMeterRequest,
  QueryMeterResponse,
} from '../models/operations/meters.js'

export class Meters {
  constructor(private readonly _client: Client) {}

  async create(
    request: CreateMeterRequest,
    options?: RequestOptions,
  ): Promise<CreateMeterResponse> {
    return unwrap(await createMeter(this._client, request, options))
  }

  async get(
    request: GetMeterRequest,
    options?: RequestOptions,
  ): Promise<GetMeterResponse> {
    return unwrap(await getMeter(this._client, request, options))
  }

  async list(
    request?: ListMetersRequest,
    options?: RequestOptions,
  ): Promise<ListMetersResponse> {
    return unwrap(await listMeters(this._client, request, options))
  }

  async update(
    request: UpdateMeterRequest,
    options?: RequestOptions,
  ): Promise<UpdateMeterResponse> {
    return unwrap(await updateMeter(this._client, request, options))
  }

  async delete(
    request: DeleteMeterRequest,
    options?: RequestOptions,
  ): Promise<DeleteMeterResponse> {
    return unwrap(await deleteMeter(this._client, request, options))
  }

  async query(
    request: QueryMeterRequest,
    options?: RequestOptions,
  ): Promise<QueryMeterResponse> {
    return unwrap(await queryMeter(this._client, request, options))
  }
}
