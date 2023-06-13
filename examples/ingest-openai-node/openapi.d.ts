import type {
  OpenAPIClient,
  Parameters,
  UnknownParamsObject,
  OperationResponse,
  AxiosRequestConfig,
} from 'openapi-client-axios'

declare namespace Components {
  namespace Schemas {
    export interface Error {
      statusCode?: number // int32
      status?: string
      code?: number // int32
      message?: string
    }
    /**
     * CloudEvents Specification JSON Schema
     */
    export interface Event {
      /**
       * Identifies the event.
       * example:
       * A234-1234-1234
       */
      id: string
      /**
       * Identifies the context in which an event happened.
       * example:
       * https://github.com/cloudevents
       */
      source: string // uri-reference
      /**
       * The version of the CloudEvents specification which the event uses.
       * example:
       * 1.0
       */
      specversion: string
      /**
       * Describes the type of event related to the originating occurrence.
       * example:
       * com.github.pull_request.opened
       */
      type: string
      /**
       * Content type of the data value. Must adhere to RFC 2046 format.
       * example:
       * application/json
       */
      datacontenttype?: 'application/json'
      /**
       * Identifies the schema that data adheres to.
       */
      dataschema?: string | null // uri
      /**
       * Describes the subject of the event in the context of the event producer (identified by source).
       * example:
       * mynewfile.jpg
       */
      subject: string | null
      /**
       * Timestamp of when the occurrence happened. Must adhere to RFC 3339.
       * example:
       * 2018-04-05T17:31:00Z
       */
      time?: string | null // date-time
      /**
       * The event payload.
       * example:
       * {"foo": "bar"}
       *
       */
      data?: {
        [name: string]: any
      }
    }
    export interface Meter {
      /**
       * example:
       * my_meter
       */
      id?: string
      /**
       * example:
       * My Meter
       */
      name?: string
      /**
       * example:
       * My Meter Description
       */
      description?: string
      /**
       * example:
       * {
       *   "my_label": "my_value"
       * }
       *
       */
      labels?: {
        [name: string]: string
      }
      /**
       * example:
       * event_type
       */
      type?: string
      /**
       * example:
       * SUM
       */
      aggregation?:
        | 'SUM'
        | 'COUNT'
        | 'MAX'
        | 'COUNT_DISTINCT'
        | 'LATEST_BY_OFFSET'
      /**
       * JSONPath expression to extract the value from the event data.
       * example:
       * $.duration_ms
       */
      valueProperty?: string
      /**
       * JSONPath expressions to extract the group by values from the event data.
       * example:
       * [
       *   "$.my_label"
       * ]
       *
       */
      groupBy?: string[]
    }
  }
}
declare namespace Paths {
  namespace GetMeters {
    namespace Responses {
      export type $200 = Components.Schemas.Meter[]
      export type Default = Components.Schemas.Error
    }
  }
  namespace GetMetersById {
    namespace Parameters {
      export type MeterId = string
    }
    export interface PathParameters {
      meterId: Parameters.MeterId
    }
    namespace Responses {
      export type $200 = Components.Schemas.Meter
      export type $404 = Components.Schemas.Error
      export type Default = Components.Schemas.Error
    }
  }
  namespace GetValuesByMeterId {
    namespace Parameters {
      export type From = string // date-time
      export type MeterId = string
      export type Subject = string
      export type To = string // date-time
    }
    export interface PathParameters {
      meterId: Parameters.MeterId
    }
    export interface QueryParameters {
      from?: Parameters.From /* date-time */
      to?: Parameters.To /* date-time */
      subject?: Parameters.Subject
    }
    namespace Responses {
      export interface $200 {
        values?: {
          subject?: string
          windowStart?: string // date-time
          windowEnd?: string // date-time
          value?: number
          groupBy?: {
            [name: string]: string
          }
        }[]
      }
      export type Default = Components.Schemas.Error
    }
  }
  namespace IngestEvents {
    export type RequestBody =
      /* CloudEvents Specification JSON Schema */ Components.Schemas.Event
    namespace Responses {
      export interface $200 {}
      export type Default = Components.Schemas.Error
    }
  }
}

export interface OperationMethods {
  /**
   * ingestEvents - Ingest events
   */
  'ingestEvents'(
    parameters?: Parameters<UnknownParamsObject> | null,
    data?: Paths.IngestEvents.RequestBody,
    config?: AxiosRequestConfig
  ): OperationResponse<Paths.IngestEvents.Responses.$200>
  /**
   * getMeters - Get meters
   */
  'getMeters'(
    parameters?: Parameters<UnknownParamsObject> | null,
    data?: any,
    config?: AxiosRequestConfig
  ): OperationResponse<Paths.GetMeters.Responses.$200>
  /**
   * getMetersById - Get meter by ID
   */
  'getMetersById'(
    parameters?: Parameters<Paths.GetMetersById.PathParameters> | null,
    data?: any,
    config?: AxiosRequestConfig
  ): OperationResponse<Paths.GetMetersById.Responses.$200>
  /**
   * getValuesByMeterId - Get meter values
   */
  'getValuesByMeterId'(
    parameters?: Parameters<
      Paths.GetValuesByMeterId.PathParameters &
        Paths.GetValuesByMeterId.QueryParameters
    > | null,
    data?: any,
    config?: AxiosRequestConfig
  ): OperationResponse<Paths.GetValuesByMeterId.Responses.$200>
}

export interface PathsDictionary {
  ['/api/v1alpha1/events']: {
    /**
     * ingestEvents - Ingest events
     */
    'post'(
      parameters?: Parameters<UnknownParamsObject> | null,
      data?: Paths.IngestEvents.RequestBody,
      config?: AxiosRequestConfig
    ): OperationResponse<Paths.IngestEvents.Responses.$200>
  }
  ['/api/v1alpha1/meters']: {
    /**
     * getMeters - Get meters
     */
    'get'(
      parameters?: Parameters<UnknownParamsObject> | null,
      data?: any,
      config?: AxiosRequestConfig
    ): OperationResponse<Paths.GetMeters.Responses.$200>
  }
  ['/api/v1alpha1/meters/{meterId}']: {
    /**
     * getMetersById - Get meter by ID
     */
    'get'(
      parameters?: Parameters<Paths.GetMetersById.PathParameters> | null,
      data?: any,
      config?: AxiosRequestConfig
    ): OperationResponse<Paths.GetMetersById.Responses.$200>
  }
  ['/api/v1alpha1/meters/{meterId}/values']: {
    /**
     * getValuesByMeterId - Get meter values
     */
    'get'(
      parameters?: Parameters<
        Paths.GetValuesByMeterId.PathParameters &
          Paths.GetValuesByMeterId.QueryParameters
      > | null,
      data?: any,
      config?: AxiosRequestConfig
    ): OperationResponse<Paths.GetValuesByMeterId.Responses.$200>
  }
}

export type Client = OpenAPIClient<OperationMethods, PathsDictionary>
