import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  return request(() => {
    const query = toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listPlansQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listPlansQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get('openmeter/plans', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listPlansResponseWire, data)
        }
        return fromWire(data, schemas.listPlansResponse)
      })
  })
}

export function createPlan(
  client: Client,
  req: CreatePlanRequest,
  options?: RequestOptions,
): Promise<Result<CreatePlanResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createPlanBody)
    if (client._options.validate) {
      assertValid(schemas.createPlanBodyWire, body)
    }
    return http(client)
      .post('openmeter/plans', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createPlanResponseWire, data)
        }
        return fromWire(data, schemas.createPlanResponse)
      })
  })
}

export function updatePlan(
  client: Client,
  req: UpdatePlanRequest,
  options?: RequestOptions,
): Promise<Result<UpdatePlanResponse>> {
  const path = `openmeter/plans/${(() => {
    if (req.planId === undefined) {
      throw new Error('missing path parameter: planId')
    }
    return encodeURIComponent(String(req.planId))
  })()}`
  return request(() => {
    const body = toWire(req.body, schemas.updatePlanBody)
    if (client._options.validate) {
      assertValid(schemas.updatePlanBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updatePlanResponseWire, data)
        }
        return fromWire(data, schemas.updatePlanResponse)
      })
  })
}

export function getPlan(
  client: Client,
  req: GetPlanRequest,
  options?: RequestOptions,
): Promise<Result<GetPlanResponse>> {
  const path = `openmeter/plans/${(() => {
    if (req.planId === undefined) {
      throw new Error('missing path parameter: planId')
    }
    return encodeURIComponent(String(req.planId))
  })()}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getPlanResponseWire, data)
        }
        return fromWire(data, schemas.getPlanResponse)
      }),
  )
}

export function deletePlan(
  client: Client,
  req: DeletePlanRequest,
  options?: RequestOptions,
): Promise<Result<DeletePlanResponse>> {
  const path = `openmeter/plans/${(() => {
    if (req.planId === undefined) {
      throw new Error('missing path parameter: planId')
    }
    return encodeURIComponent(String(req.planId))
  })()}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function archivePlan(
  client: Client,
  req: ArchivePlanRequest,
  options?: RequestOptions,
): Promise<Result<ArchivePlanResponse>> {
  const path = `openmeter/plans/${(() => {
    if (req.planId === undefined) {
      throw new Error('missing path parameter: planId')
    }
    return encodeURIComponent(String(req.planId))
  })()}/archive`
  return request(() =>
    http(client)
      .post(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.archivePlanResponseWire, data)
        }
        return fromWire(data, schemas.archivePlanResponse)
      }),
  )
}

export function publishPlan(
  client: Client,
  req: PublishPlanRequest,
  options?: RequestOptions,
): Promise<Result<PublishPlanResponse>> {
  const path = `openmeter/plans/${(() => {
    if (req.planId === undefined) {
      throw new Error('missing path parameter: planId')
    }
    return encodeURIComponent(String(req.planId))
  })()}/publish`
  return request(() =>
    http(client)
      .post(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.publishPlanResponseWire, data)
        }
        return fromWire(data, schemas.publishPlanResponse)
      }),
  )
}
