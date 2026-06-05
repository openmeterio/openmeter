import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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

export function createMeter(
  client: Client,
  req: CreateMeterRequest,
  options?: RequestOptions,
): Promise<Result<CreateMeterResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/meters', { ...options, json: req })
      .json<CreateMeterResponse>(),
  )
}

export function getMeter(
  client: Client,
  req: GetMeterRequest,
  options?: RequestOptions,
): Promise<Result<GetMeterResponse>> {
  const path = encodePath('openmeter/meters/{meterId}', { meterId: req.meterId })
  return request(() =>
    http(client)
      .get(path, options)
      .json<GetMeterResponse>(),
  )
}

export function listMeters(
  client: Client,
  req: ListMetersRequest = {},
  options?: RequestOptions,
): Promise<Result<ListMetersResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/meters', { ...options, searchParams })
      .json<ListMetersResponse>(),
  )
}

export function updateMeter(
  client: Client,
  req: UpdateMeterRequest,
  options?: RequestOptions,
): Promise<Result<UpdateMeterResponse>> {
  const path = encodePath('openmeter/meters/{meterId}', { meterId: req.meterId })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdateMeterResponse>(),
  )
}

export function deleteMeter(
  client: Client,
  req: DeleteMeterRequest,
  options?: RequestOptions,
): Promise<Result<DeleteMeterResponse>> {
  const path = encodePath('openmeter/meters/{meterId}', { meterId: req.meterId })
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function queryMeter(
  client: Client,
  req: QueryMeterRequest,
  options?: RequestOptions,
): Promise<Result<QueryMeterResponse>> {
  const path = encodePath('openmeter/meters/{meterId}/query', { meterId: req.meterId })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<QueryMeterResponse>(),
  )
}
