import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';
import http, { RefinedResponse, ResponseType } from 'k6/http';

import { QUERY_EXECUTION_TIME } from '../shared/metrics';

export interface OpenMeterClientConfig {
  baseURL: string;
  telemetryURL: string;
  apiVersion: string;
}

export interface OpenMeterResponse {
  status: number;
  body: unknown;
}

export interface OpenMeterClient {
  healthCheck(): boolean;
  queryMeter(
    meter: string,
    groupBy: string[],
    opts?: { tags?: Record<string, string>; registerMeters?: (res: RefinedResponse<ResponseType>) => void }
  ): OpenMeterResponse;
}

export const createOpenMeterClient = (config: OpenMeterClientConfig): OpenMeterClient => {
  const telemetryURL = new URL(`${config.telemetryURL}/healthz/live`);
  const baseURL = new URL(`${config.baseURL}/api/${config.apiVersion}`);
  return {
    healthCheck(): boolean {
      const res = http.get(telemetryURL.toString());
      return res.status === 200;
    },
    queryMeter(meter: string, groupBy: string[], opts): OpenMeterResponse {
      const url = new URL(`${baseURL}/meters/${meter}/query`);
      groupBy.forEach((group) => url.searchParams.append('groupBy', group));
      const res = http.get(url.toString(), { tags: opts?.tags });
      if (opts?.registerMeters) {
        opts.registerMeters(res);
      }
      QUERY_EXECUTION_TIME.add(res.timings.duration);
      return { status: res.status, body: res.json() };
    },
  };
};
