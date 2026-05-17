import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface GovernanceClientContext extends Client {

}export interface GovernanceClientOptions extends ClientOptions {
  endpoint?: string;
}export function createGovernanceClientContext(
  endpoint: string,
  options?: GovernanceClientOptions,
): GovernanceClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
