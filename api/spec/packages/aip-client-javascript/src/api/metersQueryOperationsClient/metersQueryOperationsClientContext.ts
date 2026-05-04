import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface MetersQueryOperationsClientContext extends Client {

}export interface MetersQueryOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createMetersQueryOperationsClientContext(
  endpoint: string,
  options?: MetersQueryOperationsClientOptions,
): MetersQueryOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
