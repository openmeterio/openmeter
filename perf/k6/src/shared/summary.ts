import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

import { MetricSummary } from './metrics';

export interface SummaryData {
  metrics: Record<string, MetricSummary>;
  options: unknown;
  root_group: unknown;
  state: {
    testRunDurationMs: number;
  };
}

export type SummaryHandler = (data: unknown) => Record<string, unknown>;

/**
 * @see https://grafana.com/docs/k6/latest/results-output/end-of-test/custom-summary/
 */
export const summaryHandlerFactory =
  (config: { includeMetrics: string[]; generateJSON?: boolean }) =>
  (data: SummaryData): Record<string, unknown> => {
    if (typeof data !== 'object' || data === null) {
      throw new Error('Invalid data');
    }

    const filteredMeters = Object.fromEntries(
      Object.entries(data.metrics).filter(([key]) => config.includeMetrics.includes(key))
    );

    const filename = new Date().toUTCString().replace(/[^a-zA-Z0-9]/g, '_');

    if (config.generateJSON) {
      return {
        stdout: textSummary(data as unknown, { enableColors: true }),
        [`reports/${filename}.json`]: JSON.stringify({ ...data, metrics: filteredMeters }, null, 4),
      };
    }
    return {
      stdout: textSummary(data as unknown, { enableColors: true }),
      ...(config.generateJSON
        ? { [`reports/${filename}.json`]: JSON.stringify({ ...data, metrics: filteredMeters }, null, 4) }
        : {}),
    };
  };
