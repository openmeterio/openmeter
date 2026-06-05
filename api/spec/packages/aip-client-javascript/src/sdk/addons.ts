import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listAddons,
  createAddon,
  updateAddon,
  getAddon,
  deleteAddon,
  archiveAddon,
  publishAddon,
} from '../funcs/addons.js'
import type {
  ListAddonsRequest,
  ListAddonsResponse,
  CreateAddonRequest,
  CreateAddonResponse,
  UpdateAddonRequest,
  UpdateAddonResponse,
  GetAddonRequest,
  GetAddonResponse,
  DeleteAddonRequest,
  DeleteAddonResponse,
  ArchiveAddonRequest,
  ArchiveAddonResponse,
  PublishAddonRequest,
  PublishAddonResponse,
} from '../models/operations/addons.js'

export class Addons {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListAddonsRequest,
    options?: RequestOptions,
  ): Promise<ListAddonsResponse> {
    return unwrap(await listAddons(this._client, request, options))
  }

  async create(
    request: CreateAddonRequest,
    options?: RequestOptions,
  ): Promise<CreateAddonResponse> {
    return unwrap(await createAddon(this._client, request, options))
  }

  async update(
    request: UpdateAddonRequest,
    options?: RequestOptions,
  ): Promise<UpdateAddonResponse> {
    return unwrap(await updateAddon(this._client, request, options))
  }

  async get(
    request: GetAddonRequest,
    options?: RequestOptions,
  ): Promise<GetAddonResponse> {
    return unwrap(await getAddon(this._client, request, options))
  }

  async delete(
    request: DeleteAddonRequest,
    options?: RequestOptions,
  ): Promise<DeleteAddonResponse> {
    return unwrap(await deleteAddon(this._client, request, options))
  }

  async archive(
    request: ArchiveAddonRequest,
    options?: RequestOptions,
  ): Promise<ArchiveAddonResponse> {
    return unwrap(await archiveAddon(this._client, request, options))
  }

  async publish(
    request: PublishAddonRequest,
    options?: RequestOptions,
  ): Promise<PublishAddonResponse> {
    return unwrap(await publishAddon(this._client, request, options))
  }
}
