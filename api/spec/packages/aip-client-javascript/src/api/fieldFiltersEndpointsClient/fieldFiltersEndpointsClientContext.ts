import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface FieldFiltersEndpointsClientContext extends Client {

}export interface FieldFiltersEndpointsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createFieldFiltersEndpointsClientContext(
  endpoint: string,
  options?: FieldFiltersEndpointsClientOptions,
): FieldFiltersEndpointsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
