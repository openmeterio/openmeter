/**
 * This file was auto-generated by openapi-typescript.
 * Do not make direct changes to the file.
 */

export interface paths {
  '/api/v1/events': {
    /** @description Retrieve latest raw events. */
    get: operations['listEvents']
    /** @description Ingest events */
    post: operations['ingestEvents']
  }
  '/api/v1/meters': {
    /** @description List meters */
    get: operations['listMeters']
    /** @description Create meter */
    post: operations['createMeter']
  }
  '/api/v1/meters/{meterIdOrSlug}': {
    /** @description Get meter by slugs */
    get: operations['getMeter']
    /** @description Delete meter by slug */
    delete: operations['deleteMeter']
  }
  '/api/v1/meters/{meterIdOrSlug}/query': {
    /** @description Query meter */
    get: operations['queryMeter']
  }
  '/api/v1/meters/{meterIdOrSlug}/subjects': {
    /** @description List meter subjects */
    get: operations['listMeterSubjects']
  }
  '/api/v1/portal/tokens': {
    post: operations['createPortalToken']
  }
  '/api/v1/portal/tokens/invalidate': {
    post: operations['invalidatePortalTokens']
  }
}

export type webhooks = Record<string, never>

export interface components {
  schemas: {
    /**
     * @description A Problem Details object (RFC 7807)
     * @example {
     *   "type": "urn:problem-type:bad-request",
     *   "title": "Bad Request",
     *   "status": 400,
     *   "detail": "header Content-Type has unexpected value \"application/xml\"",
     *   "instance": "urn:request:local/JMOlctsKV8-000001"
     * }
     */
    Problem: {
      /**
       * Format: uri
       * @description Type contains a URI that identifies the problem type.
       * @example urn:problem-type:bad-request
       */
      type: string
      /**
       * @description A a short, human-readable summary of the problem type.
       * @example Bad Request
       */
      title: string
      /**
       * Format: int32
       * @description The HTTP status code generated by the origin server for this occurrence of the problem.
       * @example 400
       */
      status: number
      /**
       * @description A human-readable explanation specific to this occurrence of the problem.
       * @example header Content-Type has unexpected value \"application/xml\"
       */
      detail: string
      /**
       * Format: uri
       * @description A URI reference that identifies the specific occurrence of the problem.
       * @example urn:request:local/JMOlctsKV8-000001
       */
      instance?: string
      [key: string]: unknown
    }
    /** @description CloudEvents Specification JSON Schema */
    Event: {
      /**
       * @description Identifies the event.
       * @example 5c10fade-1c9e-4d6c-8275-c52c36731d3c
       */
      id: string
      /**
       * Format: uri-reference
       * @description Identifies the context in which an event happened.
       * @example services/service-0
       */
      source: string
      /**
       * @description The version of the CloudEvents specification which the event uses.
       * @example 1.0
       */
      specversion: string
      /**
       * @description Describes the type of event related to the originating occurrence.
       * @example api_request
       */
      type: string
      /**
       * @description Content type of the data value. Must adhere to RFC 2046 format.
       * @example application/json
       * @enum {string|null}
       */
      datacontenttype?: 'application/json' | null
      /**
       * Format: uri
       * @description Identifies the schema that data adheres to.
       */
      dataschema?: string | null
      /**
       * @description Describes the subject of the event in the context of the event producer (identified by source).
       * @example customer_id
       */
      subject: string
      /**
       * Format: date-time
       * @description Timestamp of when the occurrence happened. Must adhere to RFC 3339.
       * @example 2023-01-01T01:01:01.001Z
       */
      time?: string | null
      /**
       * @description The event payload.
       * @example {
       *   "duration_ms": "12",
       *   "path": "/hello"
       * }
       */
      data?: {
        [key: string]: unknown
      }
    }
    IngestedEvent: {
      event: components['schemas']['Event']
      validationError?: string
    }
    Meter: {
      /**
       * @description A unique identifier for the meter.
       * @example 01G65Z755AFWAKHE12NY0CQ9FH
       */
      id?: string
      /**
       * @description A unique identifier for the meter.
       * @example my_meter
       */
      slug: string
      /**
       * @description A description of the meter.
       * @example My Meter Description
       */
      description?: string | null
      aggregation: components['schemas']['MeterAggregation']
      windowSize: components['schemas']['WindowSize']
      /**
       * @description The event type to aggregate.
       * @example api_request
       */
      eventType: string
      /**
       * @description JSONPath expression to extract the value from the event data.
       * @example $.duration_ms
       */
      valueProperty?: string
      /**
       * @description Named JSONPath expressions to extract the group by values from the event data.
       * @example {
       *   "method": "$.method",
       *   "path": "$.path"
       * }
       */
      groupBy?: {
        [key: string]: string
      }
    }
    /**
     * @description The aggregation type to use for the meter.
     * @enum {string}
     */
    MeterAggregation: 'SUM' | 'COUNT' | 'AVG' | 'MIN' | 'MAX'
    /**
     * @description Aggregation window size.
     * @enum {string}
     */
    WindowSize: 'MINUTE' | 'HOUR' | 'DAY'
    MeterQueryRow: {
      value: number
      /** Format: date-time */
      windowStart: string
      /** Format: date-time */
      windowEnd: string
      /** @description The subject of the meter value. */
      subject?: string | null
      groupBy?: {
        [key: string]: string
      } | null
    }
    PortalToken: {
      subject: string
      /** Format: date-time */
      expiresAt: string
      token: string
      allowedMeterSlugs?: string[]
    }
    IdOrSlug: string
  }
  responses: {
    /** @description Bad Request */
    BadRequestProblemResponse: {
      content: {
        'application/problem+json': components['schemas']['Problem']
      }
    }
    /** @description Method not allowed, feature not supported */
    MethodNotAllowedProblemResponse: {
      content: {
        'application/problem+json': components['schemas']['Problem']
      }
    }
    /** @description Not Found */
    NotFoundProblemResponse: {
      content: {
        'application/problem+json': components['schemas']['Problem']
      }
    }
    /** @description Not Implemented */
    NotImplementedProblemResponse: {
      content: {
        'application/problem+json': components['schemas']['Problem']
      }
    }
    /** @description Unexpected error */
    UnexpectedProblemResponse: {
      content: {
        'application/problem+json': components['schemas']['Problem']
      }
    }
  }
  parameters: {
    /** @description A unique identifier for the meter. */
    meterIdOrSlug: components['schemas']['IdOrSlug']
    /**
     * @description Start date-time in RFC 3339 format.
     * Inclusive.
     */
    queryFrom?: string
    /**
     * @description End date-time in RFC 3339 format.
     * Inclusive.
     */
    queryTo?: string
    /** @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group. */
    queryWindowSize?: components['schemas']['WindowSize']
    /**
     * @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
     * If not specified, the UTC timezone will be used.
     */
    queryWindowTimeZone?: string
    querySubject?: string[]
    /** @description If not specified a single aggregate will be returned for each subject and time window. */
    queryGroupBy?: string[]
  }
  requestBodies: never
  headers: never
  pathItems: never
}

export type $defs = Record<string, never>

export type external = Record<string, never>

export interface operations {
  /** @description Retrieve latest raw events. */
  listEvents: {
    parameters: {
      query?: {
        from?: components['parameters']['queryFrom']
        to?: components['parameters']['queryTo']
        /** @description Number of events to return. */
        limit?: number
      }
    }
    responses: {
      /** @description Events response */
      200: {
        content: {
          'application/json': components['schemas']['IngestedEvent'][]
        }
      }
      400: components['responses']['BadRequestProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description Ingest events */
  ingestEvents: {
    requestBody: {
      content: {
        'application/cloudevents+json': components['schemas']['Event']
        'application/cloudevents-batch+json': components['schemas']['Event'][]
      }
    }
    responses: {
      /** @description OK */
      204: {
        content: never
      }
      400: components['responses']['BadRequestProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description List meters */
  listMeters: {
    responses: {
      /** @description Meters response */
      200: {
        content: {
          'application/json': components['schemas']['Meter'][]
        }
      }
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description Create meter */
  createMeter: {
    requestBody: {
      content: {
        'application/json': components['schemas']['Meter']
      }
    }
    responses: {
      /** @description Created */
      201: {
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      400: components['responses']['BadRequestProblemResponse']
      501: components['responses']['NotImplementedProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description Get meter by slugs */
  getMeter: {
    parameters: {
      path: {
        meterIdOrSlug: components['parameters']['meterIdOrSlug']
      }
    }
    responses: {
      /** @description OK */
      200: {
        content: {
          'application/json': components['schemas']['Meter']
        }
      }
      404: components['responses']['NotFoundProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description Delete meter by slug */
  deleteMeter: {
    parameters: {
      path: {
        meterIdOrSlug: components['parameters']['meterIdOrSlug']
      }
    }
    responses: {
      /** @description No Content */
      204: {
        content: never
      }
      404: components['responses']['NotFoundProblemResponse']
      501: components['responses']['NotImplementedProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description Query meter */
  queryMeter: {
    parameters: {
      query?: {
        from?: components['parameters']['queryFrom']
        to?: components['parameters']['queryTo']
        windowSize?: components['parameters']['queryWindowSize']
        windowTimeZone?: components['parameters']['queryWindowTimeZone']
        subject?: components['parameters']['querySubject']
        groupBy?: components['parameters']['queryGroupBy']
      }
      path: {
        meterIdOrSlug: components['parameters']['meterIdOrSlug']
      }
    }
    responses: {
      /** @description OK */
      200: {
        content: {
          'application/json': {
            /** Format: date-time */
            from?: string
            /** Format: date-time */
            to?: string
            windowSize?: components['schemas']['WindowSize']
            data: components['schemas']['MeterQueryRow'][]
          }
          'text/csv': string
        }
      }
      400: components['responses']['BadRequestProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  /** @description List meter subjects */
  listMeterSubjects: {
    parameters: {
      path: {
        meterIdOrSlug: components['parameters']['meterIdOrSlug']
      }
    }
    responses: {
      /** @description OK */
      200: {
        content: {
          'application/json': string[]
        }
      }
      400: components['responses']['BadRequestProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  createPortalToken: {
    requestBody?: {
      content: {
        'application/json': components['schemas']['PortalToken']
      }
    }
    responses: {
      /** @description OK */
      200: {
        content: {
          'application/json': components['schemas']['PortalToken']
        }
      }
      400: components['responses']['BadRequestProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
  invalidatePortalTokens: {
    requestBody?: {
      content: {
        'application/json': {
          subject?: string
        }
      }
    }
    responses: {
      /** @description No Content */
      204: {
        content: never
      }
      400: components['responses']['BadRequestProblemResponse']
      default: components['responses']['UnexpectedProblemResponse']
    }
  }
}
