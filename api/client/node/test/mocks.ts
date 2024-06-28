import {
  Entitlement,
  EntitlementCreateInputs,
  EntitlementValue,
  WindowedBalanceHistory,
} from '../clients/entitlement.js'
import { Feature, FeatureCreateInputs } from '../clients/feature.js'
import {
  EntitlementGrant,
  EntitlementGrantCreateInput,
} from '../clients/grant.js'
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

export const mockCreateFeatureInput: FeatureCreateInputs = {
  key: 'ai_tokens',
  name: 'AI Tokens',
  meterSlug: 'tokens_total',
}

export const mockFeature: Feature = {
  ...mockCreateFeatureInput,
  id: 'feature-1',
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
}

export const mockCreateEntitlementInput: EntitlementCreateInputs = {
  type: 'metered',
  featureKey: mockFeature.key,
  usagePeriod: {
    interval: 'MONTH',
  },
  issueAfterReset: 10000000,
}

export const mockEntitlement: Entitlement = {
  type: 'metered',
  id: 'entitlement-1',
  featureId: mockFeature.id,
  featureKey: mockFeature.key,
  subjectKey: mockSubject.key,
  usagePeriod: {
    interval: mockCreateEntitlementInput.usagePeriod.interval,
    anchor: '2024-01-01T00:00:00Z',
  },
  currentUsagePeriod: {
    from: '2024-01-01T00:00:00Z',
    to: '2024-01-01T00:00:00Z',
  },
  issueAfterReset: mockCreateEntitlementInput.issueAfterReset,
  lastReset: '2024-01-01T00:00:00Z',
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
}

export const mockEntitlementValue: EntitlementValue = {
  hasAccess: true,
  usage: 100,
  balance: 900,
  overage: 0,
}

export const mockWindowedBalanceHistory: WindowedBalanceHistory = {
  windowedHistory: [
    {
      period: {
        from: '2024-01-01T00:00:00Z',
        to: '2024-01-01T00:00:00Z',
      },
      usage: 100,
      balanceAtStart: 100,
    },
  ],
  burndownHistory: [
    {
      period: {
        from: '2024-01-01T00:00:00Z',
        to: '2024-01-01T00:00:00Z',
      },
      usage: 100,
      overage: 25,
      balanceAtStart: 100,
      grantBalancesAtStart: {
        '01ARZ3NDEKTSV4RRFFQ69G5FAV': 100,
      },
      balanceAtEnd: 100,
      grantBalancesAtEnd: {
        '01ARZ3NDEKTSV4RRFFQ69G5FAV': 100,
      },
      grantUsages: [
        {
          grantId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
          usage: 100,
        },
      ],
    },
  ],
}

export const mockEntitlementGrantCreateInput: EntitlementGrantCreateInput = {
  amount: 100,
  priority: 1,
  effectiveAt: '2023-01-01T00:00:00Z',
  expiration: {
    duration: 'HOUR',
    count: 12,
  },
  minRolloverAmount: 100,
  maxRolloverAmount: 100,
  recurrence: {
    interval: 'MONTH',
    anchor: '2024-06-28T18:29:44.867Z',
  },
}

export const mockEntitlementGrant: EntitlementGrant = {
  ...mockEntitlementGrantCreateInput,
  recurrence: {
    interval: 'MONTH',
    anchor: '2024-06-28T18:29:44.867Z',
  },
  id: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  entitlementId: mockEntitlement.id,
  nextRecurrence: '2023-01-01T00:00:00Z',
  expiresAt: '2023-01-01T00:00:00Z',
  createdAt: '2023-01-01T00:00:00Z',
  updatedAt: '2023-01-01T00:00:00Z',
}
