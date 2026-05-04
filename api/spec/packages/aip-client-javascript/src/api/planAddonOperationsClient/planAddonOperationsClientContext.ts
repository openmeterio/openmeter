import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface PlanAddonOperationsClientContext extends Client {

}export interface PlanAddonOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createPlanAddonOperationsClientContext(
  endpoint: string,
  options?: PlanAddonOperationsClientOptions,
): PlanAddonOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
