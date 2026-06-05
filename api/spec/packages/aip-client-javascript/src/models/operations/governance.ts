import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CursorPaginationQueryPage,
  GovernanceQueryRequestInput,
  GovernanceQueryResponse,
} from '../types.js'

export interface QueryGovernanceAccessQuery {
  page?: CursorPaginationQueryPage
}

export type QueryGovernanceAccessRequest = {
  body: GovernanceQueryRequestInput
} & QueryGovernanceAccessQuery
export type QueryGovernanceAccessResponse = GovernanceQueryResponse
