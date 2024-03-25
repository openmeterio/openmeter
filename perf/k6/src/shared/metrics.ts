import { Trend } from 'k6/metrics';

/**
 * Represents the summary output of a metric.
 */
export interface MetricSummary {
  type: 'trend' | 'gauge' | 'rate' | 'counter';
  contains: string;
  values: Record<string, number>;
}

// Shared Metrics...
export const QUERY_EXECUTION_TIME = new Trend('QUERY_EXECUTION_TIME');
