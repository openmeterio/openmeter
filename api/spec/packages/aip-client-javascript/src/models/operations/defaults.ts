import type { AcceptDateStrings } from '../../lib/wire.js'
import type {
  OrganizationDefaultTaxCodes,
  UpdateOrganizationDefaultTaxCodesRequest as UpdateOrganizationDefaultTaxCodesRequestBody,
} from '../types.js'

export type GetOrganizationDefaultTaxCodesRequest = Record<string, never>
export type GetOrganizationDefaultTaxCodesResponse = OrganizationDefaultTaxCodes

export type UpdateOrganizationDefaultTaxCodesRequest =
  AcceptDateStrings<UpdateOrganizationDefaultTaxCodesRequestBody>
export type UpdateOrganizationDefaultTaxCodesResponse =
  OrganizationDefaultTaxCodes
