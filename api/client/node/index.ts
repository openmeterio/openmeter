import { OpenMeterConfig } from './clients/client.js'
import { EntitlementClient } from './clients/entitlement.js'
import { EventsClient } from './clients/event.js'
import { FeatureClient } from './clients/feature.js'
import { MetersClient } from './clients/meter.js'
import { PortalClient } from './clients/portal.js'
import { SubjectClient } from './clients/subject.js'

export { OpenMeterConfig, RequestOptions } from './clients/client.js'
export { Event, IngestedEvent, CloudEvent } from './clients/event.js'
export { Meter, MeterAggregation, WindowSize } from './clients/meter.js'

export class OpenMeter {
  public events: EventsClient
  public meters: MetersClient
  public portal: PortalClient
  public subjects: SubjectClient
  public features: FeatureClient
  public entitlements: EntitlementClient

  constructor(config: OpenMeterConfig) {
    this.events = new EventsClient(config)
    this.meters = new MetersClient(config)
    this.portal = new PortalClient(config)
    this.subjects = new SubjectClient(config)
    this.features = new FeatureClient(config)
    this.entitlements = new EntitlementClient(config)
  }
}
