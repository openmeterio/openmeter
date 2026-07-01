import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  CreateCustomerRequest,
  CreateCustomerResponse,
  GetCustomerRequest,
  GetCustomerResponse,
  ListCustomersRequest,
  ListCustomersResponse,
  UpsertCustomerRequest,
  UpsertCustomerResponse,
  DeleteCustomerRequest,
  DeleteCustomerResponse,
  GetCustomerBillingRequest,
  GetCustomerBillingResponse,
  UpdateCustomerBillingRequest,
  UpdateCustomerBillingResponse,
  UpdateCustomerBillingAppDataRequest,
  UpdateCustomerBillingAppDataResponse,
  CreateCustomerStripeCheckoutSessionRequest,
  CreateCustomerStripeCheckoutSessionResponse,
  CreateCustomerStripePortalSessionRequest,
  CreateCustomerStripePortalSessionResponse,
  CreateCreditGrantRequest,
  CreateCreditGrantResponse,
  GetCreditGrantRequest,
  GetCreditGrantResponse,
  ListCreditGrantsRequest,
  ListCreditGrantsResponse,
  GetCustomerCreditBalanceRequest,
  GetCustomerCreditBalanceResponse,
  CreateCreditAdjustmentRequest,
  CreateCreditAdjustmentResponse,
  UpdateCreditGrantExternalSettlementRequest,
  UpdateCreditGrantExternalSettlementResponse,
  ListCreditTransactionsRequest,
  ListCreditTransactionsResponse,
  ListCustomerChargesRequest,
  ListCustomerChargesResponse,
  CreateCustomerChargesRequest,
  CreateCustomerChargesResponse,
} from '../models/operations/customers.js'

export function createCustomer(
  client: Client,
  req: CreateCustomerRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createCustomerBody)
    if (client._options.validate) {
      assertValid(schemas.createCustomerBodyWire, body)
    }
    return http(client)
      .post('openmeter/customers', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createCustomerResponseWire, data)
        }
        return fromWire(data, schemas.createCustomerResponse)
      })
  })
}

export function getCustomer(
  client: Client,
  req: GetCustomerRequest,
  options?: RequestOptions,
): Promise<Result<GetCustomerResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getCustomerResponseWire, data)
        }
        return fromWire(data, schemas.getCustomerResponse)
      })
  })
}

export function listCustomers(
  client: Client,
  req: ListCustomersRequest = {},
  options?: RequestOptions,
): Promise<Result<ListCustomersResponse>> {
  return request(() => {
    const query = toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listCustomersQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listCustomersQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get('openmeter/customers', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCustomersResponseWire, data)
        }
        return fromWire(data, schemas.listCustomersResponse)
      })
  })
}

export function upsertCustomer(
  client: Client,
  req: UpsertCustomerRequest,
  options?: RequestOptions,
): Promise<Result<UpsertCustomerResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}`
    const body = toWire(req.body, schemas.upsertCustomerBody)
    if (client._options.validate) {
      assertValid(schemas.upsertCustomerBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.upsertCustomerResponseWire, data)
        }
        return fromWire(data, schemas.upsertCustomerResponse)
      })
  })
}

export function deleteCustomer(
  client: Client,
  req: DeleteCustomerRequest,
  options?: RequestOptions,
): Promise<Result<DeleteCustomerResponse>> {
  return request(async () => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}`
    await http(client).delete(path, options)
  })
}

export function getCustomerBilling(
  client: Client,
  req: GetCustomerBillingRequest,
  options?: RequestOptions,
): Promise<Result<GetCustomerBillingResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/billing`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getCustomerBillingResponseWire, data)
        }
        return fromWire(data, schemas.getCustomerBillingResponse)
      })
  })
}

export function updateCustomerBilling(
  client: Client,
  req: UpdateCustomerBillingRequest,
  options?: RequestOptions,
): Promise<Result<UpdateCustomerBillingResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/billing`
    const body = toWire(req.body, schemas.updateCustomerBillingBody)
    if (client._options.validate) {
      assertValid(schemas.updateCustomerBillingBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateCustomerBillingResponseWire, data)
        }
        return fromWire(data, schemas.updateCustomerBillingResponse)
      })
  })
}

export function updateCustomerBillingAppData(
  client: Client,
  req: UpdateCustomerBillingAppDataRequest,
  options?: RequestOptions,
): Promise<Result<UpdateCustomerBillingAppDataResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/billing/app-data`
    const body = toWire(req.body, schemas.updateCustomerBillingAppDataBody)
    if (client._options.validate) {
      assertValid(schemas.updateCustomerBillingAppDataBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateCustomerBillingAppDataResponseWire, data)
        }
        return fromWire(data, schemas.updateCustomerBillingAppDataResponse)
      })
  })
}

export function createCustomerStripeCheckoutSession(
  client: Client,
  req: CreateCustomerStripeCheckoutSessionRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerStripeCheckoutSessionResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/billing/stripe/checkout-sessions`
    const body = toWire(
      req.body,
      schemas.createCustomerStripeCheckoutSessionBody,
    )
    if (client._options.validate) {
      assertValid(schemas.createCustomerStripeCheckoutSessionBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(
            schemas.createCustomerStripeCheckoutSessionResponseWire,
            data,
          )
        }
        return fromWire(
          data,
          schemas.createCustomerStripeCheckoutSessionResponse,
        )
      })
  })
}

export function createCustomerStripePortalSession(
  client: Client,
  req: CreateCustomerStripePortalSessionRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerStripePortalSessionResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/billing/stripe/portal-sessions`
    const body = toWire(req.body, schemas.createCustomerStripePortalSessionBody)
    if (client._options.validate) {
      assertValid(schemas.createCustomerStripePortalSessionBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(
            schemas.createCustomerStripePortalSessionResponseWire,
            data,
          )
        }
        return fromWire(data, schemas.createCustomerStripePortalSessionResponse)
      })
  })
}

export function createCreditGrant(
  client: Client,
  req: CreateCreditGrantRequest,
  options?: RequestOptions,
): Promise<Result<CreateCreditGrantResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/grants`
    const body = toWire(req.body, schemas.createCreditGrantBody)
    if (client._options.validate) {
      assertValid(schemas.createCreditGrantBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createCreditGrantResponseWire, data)
        }
        return fromWire(data, schemas.createCreditGrantResponse)
      })
  })
}

export function getCreditGrant(
  client: Client,
  req: GetCreditGrantRequest,
  options?: RequestOptions,
): Promise<Result<GetCreditGrantResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/grants/${(() => {
      if (req.creditGrantId === undefined) {
        throw new Error('missing path parameter: creditGrantId')
      }
      return encodeURIComponent(String(req.creditGrantId))
    })()}`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getCreditGrantResponseWire, data)
        }
        return fromWire(data, schemas.getCreditGrantResponse)
      })
  })
}

export function listCreditGrants(
  client: Client,
  req: ListCreditGrantsRequest,
  options?: RequestOptions,
): Promise<Result<ListCreditGrantsResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/grants`
    const query = toWire(
      {
        page: req.page,
        filter: req.filter,
      },
      schemas.listCreditGrantsQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listCreditGrantsQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCreditGrantsResponseWire, data)
        }
        return fromWire(data, schemas.listCreditGrantsResponse)
      })
  })
}

export function getCustomerCreditBalance(
  client: Client,
  req: GetCustomerCreditBalanceRequest,
  options?: RequestOptions,
): Promise<Result<GetCustomerCreditBalanceResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/balance`
    const query = toWire(
      {
        timestamp: req.timestamp,
        filter: req.filter,
      },
      schemas.getCustomerCreditBalanceQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.getCustomerCreditBalanceQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getCustomerCreditBalanceResponseWire, data)
        }
        return fromWire(data, schemas.getCustomerCreditBalanceResponse)
      })
  })
}

export function createCreditAdjustment(
  client: Client,
  req: CreateCreditAdjustmentRequest,
  options?: RequestOptions,
): Promise<Result<CreateCreditAdjustmentResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/adjustments`
    const body = toWire(req.body, schemas.createCreditAdjustmentBody)
    if (client._options.validate) {
      assertValid(schemas.createCreditAdjustmentBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createCreditAdjustmentResponseWire, data)
        }
        return fromWire(data, schemas.createCreditAdjustmentResponse)
      })
  })
}

export function updateCreditGrantExternalSettlement(
  client: Client,
  req: UpdateCreditGrantExternalSettlementRequest,
  options?: RequestOptions,
): Promise<Result<UpdateCreditGrantExternalSettlementResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/grants/${(() => {
      if (req.creditGrantId === undefined) {
        throw new Error('missing path parameter: creditGrantId')
      }
      return encodeURIComponent(String(req.creditGrantId))
    })()}/settlement/external`
    const body = toWire(
      req.body,
      schemas.updateCreditGrantExternalSettlementBody,
    )
    if (client._options.validate) {
      assertValid(schemas.updateCreditGrantExternalSettlementBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(
            schemas.updateCreditGrantExternalSettlementResponseWire,
            data,
          )
        }
        return fromWire(
          data,
          schemas.updateCreditGrantExternalSettlementResponse,
        )
      })
  })
}

export function listCreditTransactions(
  client: Client,
  req: ListCreditTransactionsRequest,
  options?: RequestOptions,
): Promise<Result<ListCreditTransactionsResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/credits/transactions`
    const query = toWire(
      {
        page: req.page,
        filter: req.filter,
      },
      schemas.listCreditTransactionsQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listCreditTransactionsQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCreditTransactionsResponseWire, data)
        }
        return fromWire(data, schemas.listCreditTransactionsResponse)
      })
  })
}

export function listCustomerCharges(
  client: Client,
  req: ListCustomerChargesRequest,
  options?: RequestOptions,
): Promise<Result<ListCustomerChargesResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/charges`
    const query = toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
        expand: req.expand,
      },
      schemas.listCustomerChargesQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listCustomerChargesQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCustomerChargesResponseWire, data)
        }
        return fromWire(data, schemas.listCustomerChargesResponse)
      })
  })
}

export function createCustomerCharges(
  client: Client,
  req: CreateCustomerChargesRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerChargesResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/charges`
    const body = toWire(req.body, schemas.createCustomerChargesBody)
    if (client._options.validate) {
      assertValid(schemas.createCustomerChargesBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createCustomerChargesResponseWire, data)
        }
        return fromWire(data, schemas.createCustomerChargesResponse)
      })
  })
}
