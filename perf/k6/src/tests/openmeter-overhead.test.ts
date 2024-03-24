import { sleep } from 'k6';
import { Trend } from 'k6/metrics';

import { createClickHouseClient } from '../clickhouse/client';
import { getQueryMaterializedViewQuery } from '../clickhouse/queries/query-materialized-view.query';
import { createOpenMeterClient } from '../openmeter/client';
import { getConfig } from '../shared/config';
import { summaryHandlerFactory } from '../shared/summary';

const clickhouseTrend = new Trend('CLICKHOUSE');
const openmeterTrend = new Trend('OPENMETER');

const TEST_DURATION = '3m';

// In this case VU limit is derived from the concurrent connection limit for ClickHouse
// which by defualt is 100
const VU_LIMIT = 100;

const config = getConfig();
const clickHouseClient = createClickHouseClient({
  baseURL: config.clickhouse.baseURL,
  user: config.clickhouse.user,
  password: config.clickhouse.password,
  database: config.clickhouse.database,
});
const openMeterClient = createOpenMeterClient({
  apiVersion: 'v1',
  baseURL: config.openMeter.baseURL,
  telemetryURL: config.openMeter.telemetryURL,
});

export function setup(): void {
  if (!clickHouseClient.healthCheck()) {
    throw new Error('ClickHouse health check failed');
  }

  if (!openMeterClient.healthCheck()) {
    throw new Error('OpenMeter health check failed');
  }
}

export const options = {
  scenarios: {
    clickhouse: {
      executor: 'constant-vus',
      vus: Math.floor(VU_LIMIT / 2),
      duration: TEST_DURATION,
      exec: 'queryClickHouse',
    },
    openmeter: {
      executor: 'constant-vus',
      vus: Math.floor(VU_LIMIT / 2),
      duration: TEST_DURATION,
      exec: 'queryOpenMeter',
    },
  },
};

export function queryClickHouse(): void {
  try {
    const { status, body } = clickHouseClient.query(
      getQueryMaterializedViewQuery('openmeter.om_default_grouped_sum_meter', `sumMerge(value)`, ``),
      { tags: { query: '1' }, registerMeters: (res) => clickhouseTrend.add(res.timings.duration) }
    );
    if (!status.toString().startsWith('2')) {
      throw new Error(`Query failed: ${status} ${body}`);
    }
  } finally {
    sleep(1);
  }
}

export function queryOpenMeter(): void {
  try {
    const { body, status } = openMeterClient.queryMeter('grouped_sum_meter', ['group1', 'group2'], {
      tags: { query: '2' },
      registerMeters: (res) => openmeterTrend.add(res.timings.duration),
    });
    if (!status.toString().startsWith('2')) {
      throw new Error(`Query failed: ${status} ${JSON.stringify(body as object)}`);
    }
  } finally {
    sleep(1);
  }
}

export const handleSummary = summaryHandlerFactory({
  includeMetrics: [clickhouseTrend.name, openmeterTrend.name],
  generateJSON: config.createJSONreport,
});
