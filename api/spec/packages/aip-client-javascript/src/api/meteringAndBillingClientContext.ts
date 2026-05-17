import {
  type BearerTokenCredential,
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface MeteringAndBillingClientContext extends Client {

}export interface MeteringAndBillingClientOptions extends ClientOptions {
  endpoint?: string;
}export function createMeteringAndBillingClientContext(
  endpoint: "https://global.api.konghq.com/v3" | "https://in.api.konghq.com/v3" | "https://me.api.konghq.com/v3" | "https://au.api.konghq.com/v3" | "https://eu.api.konghq.com/v3" | "https://us.api.konghq.com/v3" | string,
  credential: BearerTokenCredential,
  options?: MeteringAndBillingClientOptions,
): MeteringAndBillingClientContext {
  const params: Record<string, any> = {};
  const resolvedEndpoint = "https://global.api.konghq.com/v3".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options,credential,authSchemes: [{
      kind: "http",
      scheme: "bearer"
    },
    {
      kind: "http",
      scheme: "bearer"
    },
    {
      kind: "http",
      scheme: "bearer"
    }]
  })
}
