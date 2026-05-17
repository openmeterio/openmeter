import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface PlanOperationsClientContext extends Client {

}export interface PlanOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createPlanOperationsClientContext(
  endpoint: string,
  options?: PlanOperationsClientOptions,
): PlanOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
