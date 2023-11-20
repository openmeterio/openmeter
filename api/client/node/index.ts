import { OpenMeterConfig } from './clients/client.js'
import { EventsClient } from './clients/event.js'
import { MetersClient } from './clients/meter.js'
import { PortalClient } from './clients/portal.js'

export { OpenMeterConfig, RequestOptions } from './clients/client.js'
export { Event, IngestedEvent } from './clients/event.js'
export { Meter, MeterAggregation, WindowSize } from './clients/meter.js'

export { createOpenAIStreamCallback } from './next.js'

export class OpenMeter {
  public events: EventsClient
  public meters: MetersClient
  public portal: PortalClient

  constructor(config: OpenMeterConfig) {
    this.events = new EventsClient(config)
    this.meters = new MetersClient(config)
    this.portal = new PortalClient(config)
  }
}
