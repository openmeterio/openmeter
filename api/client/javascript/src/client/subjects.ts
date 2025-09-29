import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { operations, paths, SubjectUpsert } from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Subjects
 * @description Subjects are entities that consume resources you wish to meter. These can range from users, servers, and services to devices. The design of subjects is intentionally generic, enabling flexible application across various metering scenarios. Meters are aggregating events for each subject..
 */
export class Subjects {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Upsert one or multiple subjects
   * If the subject does not exist, it will be created, otherwise it will be updated.
   *
   * @param subjects - The subjects to upsert
   * @param signal - An optional abort signal
   * @returns The upserted subjects
   */
  public async upsert(
    subjects: SubjectUpsert | SubjectUpsert[],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/subjects', {
      body: Array.isArray(subjects) ? subjects : [subjects],
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a subject by ID or key
   * @param idOrKey - The ID or key of the subject
   * @param signal - An optional abort signal
   * @returns The subject
   */
  public async get(
    idOrKey: operations['getSubject']['parameters']['path']['subjectIdOrKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/subjects/{subjectIdOrKey}', {
      params: {
        path: {
          subjectIdOrKey: idOrKey,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List subjects
   * @param signal - An optional abort signal
   * @returns The subjects
   */
  public async list(options?: RequestOptions) {
    const resp = await this.client.GET('/api/v1/subjects', {
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a subject by ID or key
   * @param idOrKey - The ID or key of the subject
   * @param signal - An optional abort signal
   * @returns The deleted subject
   */
  public async delete(
    idOrKey: operations['deleteSubject']['parameters']['path']['subjectIdOrKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE('/api/v1/subjects/{subjectIdOrKey}', {
      params: {
        path: {
          subjectIdOrKey: idOrKey,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}
