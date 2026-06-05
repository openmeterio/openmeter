import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listPlans,
  createPlan,
  updatePlan,
  getPlan,
  deletePlan,
  archivePlan,
  publishPlan,
} from '../funcs/plans.js'
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

export class Plans {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListPlansRequest,
    options?: RequestOptions,
  ): Promise<ListPlansResponse> {
    return unwrap(await listPlans(this._client, request, options))
  }

  async create(
    request: CreatePlanRequest,
    options?: RequestOptions,
  ): Promise<CreatePlanResponse> {
    return unwrap(await createPlan(this._client, request, options))
  }

  async update(
    request: UpdatePlanRequest,
    options?: RequestOptions,
  ): Promise<UpdatePlanResponse> {
    return unwrap(await updatePlan(this._client, request, options))
  }

  async get(
    request: GetPlanRequest,
    options?: RequestOptions,
  ): Promise<GetPlanResponse> {
    return unwrap(await getPlan(this._client, request, options))
  }

  async delete(
    request: DeletePlanRequest,
    options?: RequestOptions,
  ): Promise<DeletePlanResponse> {
    return unwrap(await deletePlan(this._client, request, options))
  }

  async archive(
    request: ArchivePlanRequest,
    options?: RequestOptions,
  ): Promise<ArchivePlanResponse> {
    return unwrap(await archivePlan(this._client, request, options))
  }

  async publish(
    request: PublishPlanRequest,
    options?: RequestOptions,
  ): Promise<PublishPlanResponse> {
    return unwrap(await publishPlan(this._client, request, options))
  }
}
