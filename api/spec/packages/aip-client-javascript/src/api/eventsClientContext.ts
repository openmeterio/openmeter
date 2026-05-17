import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface EventsClientContext extends Client {

}export interface EventsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createEventsClientContext(
  endpoint: string,
  options?: EventsClientOptions,
): EventsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
