import { Meter, WindowSize } from "../index.js";

export const mockMeter: Meter = {
    slug: "m1",
    aggregation: "SUM",
    eventType: "api_requests",
    valueProperty: "$.duration_ms",
    windowSize: WindowSize.HOUR,
    groupBy: {
        method: '$.method',
        path: '$.path',
    }
}

export const mockMeterValue = {
    subject: 'customer-1',
    windowStart: '2023-01-01T01:00:00.001Z',
    windowEnd: '2023-01-01T01:00:00.001Z',
    value: 1,
    groupBy: {
        method: 'GET'
    }
}
