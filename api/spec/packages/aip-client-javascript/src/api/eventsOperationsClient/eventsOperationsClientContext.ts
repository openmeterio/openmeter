import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface EventsOperationsClientContext extends Client {

}export interface EventsOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createEventsOperationsClientContext(
  endpoint: string,
  options?: EventsOperationsClientOptions,
): EventsOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
