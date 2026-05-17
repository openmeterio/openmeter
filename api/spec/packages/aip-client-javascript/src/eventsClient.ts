import {
  createEventsClientContext,
  type EventsClientContext,
  type EventsClientOptions,
} from "./api/eventsClientContext.js";
import {
  createEventsOperationsClientContext,
  type EventsOperationsClientContext,
  type EventsOperationsClientOptions,
} from "./api/eventsOperationsClient/eventsOperationsClientContext.js";
import {
  ingestEvent,
  type IngestEventOptions,
  ingestEvents,
  ingestEventsJson,
  type IngestEventsJsonOptions,
  type IngestEventsOptions,
  list,
  type ListOptions,
} from "./api/eventsOperationsClient/eventsOperationsClientOperations.js";
import type { MeteringEvent } from "./models/models.js";

export class EventsClient {
  #context: EventsClientContext
  eventsOperationsClient: EventsOperationsClient
  constructor(endpoint: string, options?: EventsClientOptions) {
    this.#context = createEventsClientContext(endpoint, options);
    this.eventsOperationsClient = new EventsOperationsClient(endpoint, options);
  }
}
export class EventsOperationsClient {
  #context: EventsOperationsClientContext
  constructor(endpoint: string, options?: EventsOperationsClientOptions) {
    this.#context = createEventsOperationsClientContext(endpoint, options);

  }
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async ingestEvent(body: MeteringEvent, options?: IngestEventOptions) {
    return ingestEvent(this.#context, body, options);
  };
  async ingestEvents(
    body: Array<MeteringEvent>,
    options?: IngestEventsOptions,
  ) {
    return ingestEvents(this.#context, body, options);
  };
  async ingestEventsJson(
    body: MeteringEvent | Array<MeteringEvent>,
    options?: IngestEventsJsonOptions,
  ) {
    return ingestEventsJson(this.#context, body, options);
  }
}
