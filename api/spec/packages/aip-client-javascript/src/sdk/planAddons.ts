import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listPlanAddons,
  createPlanAddon,
  getPlanAddon,
  updatePlanAddon,
  deletePlanAddon,
} from '../funcs/planAddons.js'
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

export class PlanAddons {
  constructor(private readonly _client: Client) {}

  async list(
    request: ListPlanAddonsRequest,
    options?: RequestOptions,
  ): Promise<ListPlanAddonsResponse> {
    return unwrap(await listPlanAddons(this._client, request, options))
  }

  async create(
    request: CreatePlanAddonRequest,
    options?: RequestOptions,
  ): Promise<CreatePlanAddonResponse> {
    return unwrap(await createPlanAddon(this._client, request, options))
  }

  async get(
    request: GetPlanAddonRequest,
    options?: RequestOptions,
  ): Promise<GetPlanAddonResponse> {
    return unwrap(await getPlanAddon(this._client, request, options))
  }

  async update(
    request: UpdatePlanAddonRequest,
    options?: RequestOptions,
  ): Promise<UpdatePlanAddonResponse> {
    return unwrap(await updatePlanAddon(this._client, request, options))
  }

  async delete(
    request: DeletePlanAddonRequest,
    options?: RequestOptions,
  ): Promise<DeletePlanAddonResponse> {
    return unwrap(await deletePlanAddon(this._client, request, options))
  }
}
