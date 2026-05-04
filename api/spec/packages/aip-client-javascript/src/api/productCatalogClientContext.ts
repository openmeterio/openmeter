import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface ProductCatalogClientContext extends Client {

}export interface ProductCatalogClientOptions extends ClientOptions {
  endpoint?: string;
}export function createProductCatalogClientContext(
  endpoint: string,
  options?: ProductCatalogClientOptions,
): ProductCatalogClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
