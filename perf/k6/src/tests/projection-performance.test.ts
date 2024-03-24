import { sleep } from 'k6';
import { Trend } from 'k6/metrics';

import { createClickHouseClient } from '../clickhouse/client';
import { getQueryMaterializedViewQuery } from '../clickhouse/queries/query-materialized-view.query';
import { getQueryGroupedProjectionQuery } from '../clickhouse/queries/query-projection.query';
import { setupProjections, validateQueriesUseProjections } from '../clickhouse/setup';
import { getConfig } from '../shared/config';
import { summaryHandlerFactory } from '../shared/summary';

const materialisedTrend = new Trend('MATERIALISED');
const projectionTrend = new Trend('PROJECTION');

const GROUP1_VALUES = ['group1_value1', 'group1_value2', 'group1_value3'];
const GROUP2_VALUES = ['group2_value1', 'group2_value2', 'group2_value3'];

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

export function setup(): void {
  if (!clickHouseClient.healthCheck()) {
    throw new Error('ClickHouse health check failed');
  }

  setupProjections(clickHouseClient);
  validateQueriesUseProjections(clickHouseClient);
}

export const options = {
  scenarios: {
    materialised: {
      executor: 'constant-vus',
      vus: Math.floor(VU_LIMIT / 2),
      duration: TEST_DURATION,
      exec: 'queryMaterializedView',
    },
    projection: {
      executor: 'constant-vus',
      vus: Math.floor(VU_LIMIT / 2),
      duration: TEST_DURATION,
      exec: 'queryProjection',
    },
  },
};

export function queryMaterializedView(): void {
  try {
    const group1 = GROUP1_VALUES[Math.floor(Math.random() * GROUP1_VALUES.length)];
    const group2 = GROUP2_VALUES[Math.floor(Math.random() * GROUP2_VALUES.length)];
    const { status, body } = clickHouseClient.query(
      getQueryMaterializedViewQuery(
        'openmeter.om_default_grouped_sum_meter',
        `sumMerge(value)`,
        `WHERE group1 = '${group1}' AND group2 = '${group2}'`
      ),
      { tags: { query: '1' }, registerMeters: (res) => materialisedTrend.add(res.timings.duration) }
    );
    if (!status.toString().startsWith('2')) {
      throw new Error(`Query failed: ${status} ${body}`);
    }
  } finally {
    sleep(1);
  }
}

export function queryProjection(): void {
  try {
    const group1 = GROUP1_VALUES[Math.floor(Math.random() * GROUP1_VALUES.length)];
    const group2 = GROUP2_VALUES[Math.floor(Math.random() * GROUP2_VALUES.length)];
    clickHouseClient.query(
      getQueryGroupedProjectionQuery(
        'openmeter.om_events',
        `sumState(cast(JSON_VALUE(data, '$.value'),'Float64'))`,
        `sumMerge(value)`,
        `WHERE group1 = '${group1}' AND group2 = '${group2}'`
      ),
      { tags: { query: '2' }, registerMeters: (res) => projectionTrend.add(res.timings.duration) }
    );
  } finally {
    sleep(1);
  }
}

export const handleSummary = summaryHandlerFactory({
  includeMetrics: [materialisedTrend.name, projectionTrend.name],
  generateJSON: config.createJSONreport,
});
