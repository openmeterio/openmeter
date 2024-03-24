import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';
import { type bytes } from 'k6';
import http, { RefinedResponse, ResponseType } from 'k6/http';

import { QUERY_EXECUTION_TIME } from '../shared/metrics';

export interface ClickHouseConfig {
  baseURL: string;
  user: string;
  password: string;
  database: string;
}

export interface ClickHouseResponse {
  status: number;
  body: string;
  summary: {
    read_rows: number;
    read_bytes: number;
    written_rows: number;
    written_bytes: number;
    total_rows_to_read: number;
    result_rows: number;
    result_bytes: number;
    elapsed_ns: number;
  };
}

export interface ClickHouseClient {
  query(
    query: string,
    opts?: { tags?: Record<string, string>; registerMeters?: (res: RefinedResponse<ResponseType>) => void }
  ): ClickHouseResponse;
  healthCheck(): boolean;
}

const CLICKHOUSE_SUMMARY_HEADER = 'X-Clickhouse-Summary';

export const createClickHouseClient = (config: ClickHouseConfig): ClickHouseClient => {
  function parseBody(body: string | bytes | null): string {
    if (body === null) {
      return '';
    }
    if (typeof body === 'string') {
      return body;
    }
    return Buffer.from(body).toString('utf-8');
  }

  return {
    query(q, opts): ClickHouseResponse {
      const url = new URL(`${config.baseURL}`);
      url.searchParams.append('database', config.database);

      const response = http.post(url.toString(), q, {
        headers: {
          'X-ClickHouse-User': config.user,
          'X-ClickHouse-Key': config.password,
        },
        tags: opts?.tags,
      });
      const summary = JSON.parse(response.headers[CLICKHOUSE_SUMMARY_HEADER]);
      QUERY_EXECUTION_TIME.add(response.timings.duration);
      if (opts?.registerMeters) {
        opts.registerMeters(response);
      }
      return {
        status: response.status,
        body: parseBody(response.body),
        summary: {
          read_rows: parseFloat(summary.read_rows),
          read_bytes: parseFloat(summary.read_bytes),
          written_rows: parseFloat(summary.written_rows),
          written_bytes: parseFloat(summary.written_bytes),
          total_rows_to_read: parseFloat(summary.total_rows_to_read),
          result_rows: parseFloat(summary.result_rows),
          result_bytes: parseFloat(summary.result_bytes),
          elapsed_ns: parseFloat(summary.elapsed_ns),
        },
      };
    },
    healthCheck() {
      const url = new URL(`${config.baseURL}/ping`);
      const response = http.get(url.toString(), {
        headers: {
          'X-ClickHouse-User': config.user,
          'X-ClickHouse-Key': config.password,
        },
      });
      return response.status === 200 && response.body === 'Ok.\n';
    },
  };
};
