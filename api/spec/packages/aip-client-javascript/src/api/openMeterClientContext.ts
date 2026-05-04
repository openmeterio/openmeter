import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface OpenMeterClientContext extends Client {

}export interface OpenMeterClientOptions extends ClientOptions {
  endpoint?: string;
}export function createOpenMeterClientContext(
  endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
  options?: OpenMeterClientOptions,
): OpenMeterClientContext {
  const params: Record<string, any> = {
    port: options?.port ?? "8888"
  };
  const resolvedEndpoint = "http://localhost:{port}/api/v3".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
