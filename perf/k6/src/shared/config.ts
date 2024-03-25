export interface Config {
  clickhouse: {
    baseURL: string;
    user: string;
    password: string;
    database: string;
  };
  openMeter: {
    baseURL: string;
    telemetryURL: string;
  };
  createJSONreport: boolean;
}

export const getConfig = (): Config => {
  return {
    clickhouse: {
      baseURL: __ENV.CLICKHOUSE_BASE_URL || 'http://127.0.0.1:8123',
      user: __ENV.CLICKHOUSE_USER || 'default',
      password: __ENV.CLICKHOUSE_PASSWORD || 'default',
      database: __ENV.CLICKHOUSE_DATABASE || 'openmeter',
    },
    openMeter: {
      baseURL: __ENV.OPENMETER_BASE_URL || 'http://127.0.0.1:8888',
      telemetryURL: __ENV.OPENMETER_TELEMETRY_URL || 'http://127.0.0.1:10000',
    },
    createJSONreport: __ENV.CREATE_JSON_REPORT === 'true',
  };
};
