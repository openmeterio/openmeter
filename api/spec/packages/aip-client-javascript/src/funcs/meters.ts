import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  QueryMeterCsvRequest,
  QueryMeterCsvResponse,
} from '../models/operations/meters.js'

export function createMeter(
  client: Client,
  req: CreateMeterRequest,
  options?: RequestOptions,
): Promise<Result<CreateMeterResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createMeterBody)
    if (client._options.validate) {
      assertValid(schemas.createMeterBodyWire, body)
    }
    return http(client)
      .post('openmeter/meters', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createMeterResponseWire, data)
        }
        return fromWire(data, schemas.createMeterResponse)
      })
  })
}

export function getMeter(
  client: Client,
  req: GetMeterRequest,
  options?: RequestOptions,
): Promise<Result<GetMeterResponse>> {
  const path = `openmeter/meters/${(() => {
    if (req.meterId === undefined) {
      throw new Error('missing path parameter: meterId')
    }
    return encodeURIComponent(String(req.meterId))
  })()}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getMeterResponseWire, data)
        }
        return fromWire(data, schemas.getMeterResponse)
      }),
  )
}

export function listMeters(
  client: Client,
  req: ListMetersRequest = {},
  options?: RequestOptions,
): Promise<Result<ListMetersResponse>> {
  return request(() => {
    const query = toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listMetersQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listMetersQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get('openmeter/meters', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listMetersResponseWire, data)
        }
        return fromWire(data, schemas.listMetersResponse)
      })
  })
}

export function updateMeter(
  client: Client,
  req: UpdateMeterRequest,
  options?: RequestOptions,
): Promise<Result<UpdateMeterResponse>> {
  const path = `openmeter/meters/${(() => {
    if (req.meterId === undefined) {
      throw new Error('missing path parameter: meterId')
    }
    return encodeURIComponent(String(req.meterId))
  })()}`
  return request(() => {
    const body = toWire(req.body, schemas.updateMeterBody)
    if (client._options.validate) {
      assertValid(schemas.updateMeterBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateMeterResponseWire, data)
        }
        return fromWire(data, schemas.updateMeterResponse)
      })
  })
}

export function deleteMeter(
  client: Client,
  req: DeleteMeterRequest,
  options?: RequestOptions,
): Promise<Result<DeleteMeterResponse>> {
  const path = `openmeter/meters/${(() => {
    if (req.meterId === undefined) {
      throw new Error('missing path parameter: meterId')
    }
    return encodeURIComponent(String(req.meterId))
  })()}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function queryMeter(
  client: Client,
  req: QueryMeterRequest,
  options?: RequestOptions,
): Promise<Result<QueryMeterResponse>> {
  const path = `openmeter/meters/${(() => {
    if (req.meterId === undefined) {
      throw new Error('missing path parameter: meterId')
    }
    return encodeURIComponent(String(req.meterId))
  })()}/query`
  return request(() => {
    const body = toWire(req.body, schemas.queryMeterBody)
    if (client._options.validate) {
      assertValid(schemas.queryMeterBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.queryMeterResponseWire, data)
        }
        return fromWire(data, schemas.queryMeterResponse)
      })
  })
}

export function queryMeterCsv(
  client: Client,
  req: QueryMeterCsvRequest,
  options?: RequestOptions,
): Promise<Result<QueryMeterCsvResponse>> {
  const headers = new Headers(options?.headers as HeadersInit | undefined)
  headers.set('accept', 'text/csv')
  const path = `openmeter/meters/${(() => {
    if (req.meterId === undefined) {
      throw new Error('missing path parameter: meterId')
    }
    return encodeURIComponent(String(req.meterId))
  })()}/query`
  return request(() => {
    const body = toWire(req.body, schemas.queryMeterCsvBody)
    if (client._options.validate) {
      assertValid(schemas.queryMeterCsvBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body, headers })
      .text()
  })
}
