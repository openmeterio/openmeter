import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  return request(() => {
    const path = `openmeter/plans/${(() => {
      if (req.planId === undefined) {
        throw new Error('missing path parameter: planId')
      }
      return encodeURIComponent(String(req.planId))
    })()}/addons`
    const query = toWire(
      {
        page: req.page,
      },
      schemas.listPlanAddonsQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listPlanAddonsQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listPlanAddonsResponseWire, data)
        }
        return fromWire(data, schemas.listPlanAddonsResponse)
      })
  })
}

export function createPlanAddon(
  client: Client,
  req: CreatePlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<CreatePlanAddonResponse>> {
  return request(() => {
    const path = `openmeter/plans/${(() => {
      if (req.planId === undefined) {
        throw new Error('missing path parameter: planId')
      }
      return encodeURIComponent(String(req.planId))
    })()}/addons`
    const body = toWire(req.body, schemas.createPlanAddonBody)
    if (client._options.validate) {
      assertValid(schemas.createPlanAddonBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createPlanAddonResponseWire, data)
        }
        return fromWire(data, schemas.createPlanAddonResponse)
      })
  })
}

export function getPlanAddon(
  client: Client,
  req: GetPlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<GetPlanAddonResponse>> {
  return request(() => {
    const path = `openmeter/plans/${(() => {
      if (req.planId === undefined) {
        throw new Error('missing path parameter: planId')
      }
      return encodeURIComponent(String(req.planId))
    })()}/addons/${(() => {
      if (req.planAddonId === undefined) {
        throw new Error('missing path parameter: planAddonId')
      }
      return encodeURIComponent(String(req.planAddonId))
    })()}`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getPlanAddonResponseWire, data)
        }
        return fromWire(data, schemas.getPlanAddonResponse)
      })
  })
}

export function updatePlanAddon(
  client: Client,
  req: UpdatePlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<UpdatePlanAddonResponse>> {
  return request(() => {
    const path = `openmeter/plans/${(() => {
      if (req.planId === undefined) {
        throw new Error('missing path parameter: planId')
      }
      return encodeURIComponent(String(req.planId))
    })()}/addons/${(() => {
      if (req.planAddonId === undefined) {
        throw new Error('missing path parameter: planAddonId')
      }
      return encodeURIComponent(String(req.planAddonId))
    })()}`
    const body = toWire(req.body, schemas.updatePlanAddonBody)
    if (client._options.validate) {
      assertValid(schemas.updatePlanAddonBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updatePlanAddonResponseWire, data)
        }
        return fromWire(data, schemas.updatePlanAddonResponse)
      })
  })
}

export function deletePlanAddon(
  client: Client,
  req: DeletePlanAddonRequest,
  options?: RequestOptions,
): Promise<Result<DeletePlanAddonResponse>> {
  return request(async () => {
    const path = `openmeter/plans/${(() => {
      if (req.planId === undefined) {
        throw new Error('missing path parameter: planId')
      }
      return encodeURIComponent(String(req.planId))
    })()}/addons/${(() => {
      if (req.planAddonId === undefined) {
        throw new Error('missing path parameter: planAddonId')
      }
      return encodeURIComponent(String(req.planAddonId))
    })()}`
    await http(client).delete(path, options)
  })
}
