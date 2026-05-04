import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface OrganizationDefaultTaxCodesOperationsClientContext extends Client {

}export interface OrganizationDefaultTaxCodesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createOrganizationDefaultTaxCodesOperationsClientContext(
  endpoint: string,
  options?: OrganizationDefaultTaxCodesOperationsClientOptions,
): OrganizationDefaultTaxCodesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
