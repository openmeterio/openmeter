import { ClickHouseClient, ClickHouseResponse } from './client';
import { getCreateGroupedProjectionQuery, getMaterialiseProjectionQuery } from './queries/create-projection.query';
import { getQueryGroupedProjectionQuery } from './queries/query-projection.query';

function throwIfFailed(res: ClickHouseResponse): ClickHouseResponse {
  if (res.status.toString().startsWith('2')) {
    return res;
  }
  throw new Error(`Query failed: ${res.status} ${res.body}`);
}

function throwIfProjectionNotUsed(res: ClickHouseResponse, projectionName: string): ClickHouseResponse {
  if (!res.body.includes(`ReadFromMergeTree (${projectionName})`)) {
    throw new Error(`Projection ${projectionName} is not used: ${res.body}`);
  }
  return res;
}

/**
 * Safely creates and materializes projections for the `openmeter.om_events` table.
 */
export const setupProjections = (clickHouseClient: ClickHouseClient): void => {
  throwIfFailed(
    clickHouseClient.query(getCreateGroupedProjectionQuery('openmeter.om_events', 'grouped_count_proj', 'count(*)'))
  );
  throwIfFailed(clickHouseClient.query(getMaterialiseProjectionQuery('openmeter.om_events', 'grouped_count_proj')));
  throwIfFailed(
    clickHouseClient.query(
      getCreateGroupedProjectionQuery(
        'openmeter.om_events',
        'grouped_sum_proj',
        `sumState(cast(JSON_VALUE(data, '$.value'),'Float64'))`
      )
    )
  );
  throwIfFailed(clickHouseClient.query(getMaterialiseProjectionQuery('openmeter.om_events', 'grouped_sum_proj')));
  throwIfFailed(
    clickHouseClient.query(
      getCreateGroupedProjectionQuery(
        'openmeter.om_events',
        'grouped_avg_proj',
        `avgState(cast(JSON_VALUE(data, '$.value'),'Float64'))`
      )
    )
  );
  throwIfFailed(clickHouseClient.query(getMaterialiseProjectionQuery('openmeter.om_events', 'grouped_avg_proj')));
};

export const validateQueriesUseProjections = (clickHouseClient: ClickHouseClient): void => {
  throwIfProjectionNotUsed(
    throwIfFailed(
      clickHouseClient.query(
        `EXPLAIN indexes = 1 ${getQueryGroupedProjectionQuery(
          'openmeter.om_events',
          `sumState(cast(JSON_VALUE(data, '$.value'),'Float64'))`,
          `sumMerge(value)`,
          ``
        )}`
      )
    ),
    'grouped_sum_proj'
  );
  throwIfProjectionNotUsed(
    throwIfFailed(
      clickHouseClient.query(
        `EXPLAIN indexes = 1 ${getQueryGroupedProjectionQuery(
          'openmeter.om_events',
          `avgState(cast(JSON_VALUE(data, '$.value'),'Float64'))`,
          `avgMerge(value)`,
          ``
        )}`
      )
    ),
    'grouped_avg_proj'
  );
  throwIfProjectionNotUsed(
    throwIfFailed(
      clickHouseClient.query(
        `EXPLAIN indexes = 1 ${getQueryGroupedProjectionQuery('openmeter.om_events', `count(*)`, `count(*)`, ``)}`
      )
    ),
    'grouped_count_proj'
  );
};
