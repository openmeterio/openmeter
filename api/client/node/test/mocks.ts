import { Subject } from '../clients/subject.js'
import { Event, Meter, WindowSize } from '../index.js'

export const mockEvent: Event = {
  specversion: '1.0',
  id: 'id-1',
  source: 'my-app',
  type: 'my-type',
  subject: 'my-awesome-user-id',
  time: new Date('2023-01-01'),
  data: {
    api_calls: 1,
  },
}

export const mockMeter: Meter = {
  slug: 'm1',
  aggregation: 'SUM',
  eventType: 'api_requests',
  valueProperty: '$.duration_ms',
  windowSize: WindowSize.HOUR,
  groupBy: {
    method: '$.method',
    path: '$.path',
  },
}

export const mockMeterValue = {
  subject: 'customer-1',
  windowStart: '2023-01-01T01:00:00.001Z',
  windowEnd: '2023-01-01T01:00:00.001Z',
  value: 1,
  groupBy: {
    method: 'GET',
  },
}

export const mockSubject: Subject = {
  id: 'abcde',
  key: 'customer-1',
  displayName: 'Customer 1',
  metadata: {
    foo: 'bar',
  },
}
