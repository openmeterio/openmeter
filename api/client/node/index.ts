import { OpenMeterConfig } from './clients/client.js'
import { EventsClient } from './clients/event.js'
import { MetersClient } from './clients/meter.js'

export { OpenMeterConfig, RequestOptions } from './clients/client.js'
export { Event } from './clients/event.js'
export { Meter, MeterAggregation, WindowSize } from './clients/meter.js'

export { createOpenAIStreamCallback } from './next.js'

export class OpenMeter {
  public events: EventsClient
  public meters: MetersClient

  constructor(config: OpenMeterConfig) {
    this.events = new EventsClient(config)
    this.meters = new MetersClient(config)
  }
}
