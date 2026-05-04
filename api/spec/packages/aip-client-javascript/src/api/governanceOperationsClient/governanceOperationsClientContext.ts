import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface GovernanceOperationsClientContext extends Client {

}export interface GovernanceOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createGovernanceOperationsClientContext(
  endpoint: string,
  options?: GovernanceOperationsClientOptions,
): GovernanceOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
