import {
  createCustomerBillingOperationsClientContext,
  type CustomerBillingOperationsClientContext,
  type CustomerBillingOperationsClientOptions,
} from "./api/customerBillingOperationsClient/customerBillingOperationsClientContext.js";
import {
  createCheckoutSession,
  type CreateCheckoutSessionOptions,
  createPortalSession,
  type CreatePortalSessionOptions,
  get as get_2,
  type GetOptions as GetOptions_2,
  upsert as upsert_2,
  upsertAppData,
  type UpsertAppDataOptions,
  type UpsertOptions as UpsertOptions_2,
} from "./api/customerBillingOperationsClient/customerBillingOperationsClientOperations.js";
import {
  createCustomerChargesOperationsClientContext,
  type CustomerChargesOperationsClientContext,
  type CustomerChargesOperationsClientOptions,
} from "./api/customerChargesOperationsClient/customerChargesOperationsClientContext.js";
import {
  list as list_4,
  type ListOptions as ListOptions_4,
} from "./api/customerChargesOperationsClient/customerChargesOperationsClientOperations.js";
import {
  createCustomerCreditAdjustmentsOperationsClientContext,
  type CustomerCreditAdjustmentsOperationsClientContext,
  type CustomerCreditAdjustmentsOperationsClientOptions,
} from "./api/customerCreditAdjustmentsOperationsClient/customerCreditAdjustmentsOperationsClientContext.js";
import {
  create as create_3,
  type CreateOptions as CreateOptions_3,
} from "./api/customerCreditAdjustmentsOperationsClient/customerCreditAdjustmentsOperationsClientOperations.js";
import {
  createCustomerCreditBalancesOperationsClientContext,
  type CustomerCreditBalancesOperationsClientContext,
  type CustomerCreditBalancesOperationsClientOptions,
} from "./api/customerCreditBalancesOperationsClient/customerCreditBalancesOperationsClientContext.js";
import {
  get as get_4,
  type GetOptions as GetOptions_4,
} from "./api/customerCreditBalancesOperationsClient/customerCreditBalancesOperationsClientOperations.js";
import {
  createCustomerCreditGrantExternalSettlementOperationsClientContext,
  type CustomerCreditGrantExternalSettlementOperationsClientContext,
  type CustomerCreditGrantExternalSettlementOperationsClientOptions,
} from "./api/customerCreditGrantExternalSettlementOperationsClient/customerCreditGrantExternalSettlementOperationsClientContext.js";
import {
  updateExternalSettlement,
  type UpdateExternalSettlementOptions,
} from "./api/customerCreditGrantExternalSettlementOperationsClient/customerCreditGrantExternalSettlementOperationsClientOperations.js";
import {
  createCustomerCreditGrantsOperationsClientContext,
  type CustomerCreditGrantsOperationsClientContext,
  type CustomerCreditGrantsOperationsClientOptions,
} from "./api/customerCreditGrantsOperationsClient/customerCreditGrantsOperationsClientContext.js";
import {
  create as create_2,
  type CreateOptions as CreateOptions_2,
  get as get_3,
  type GetOptions as GetOptions_3,
  list as list_2,
  type ListOptions as ListOptions_2,
} from "./api/customerCreditGrantsOperationsClient/customerCreditGrantsOperationsClientOperations.js";
import {
  createCustomerCreditTransactionOperationsClientContext,
  type CustomerCreditTransactionOperationsClientContext,
  type CustomerCreditTransactionOperationsClientOptions,
} from "./api/customerCreditTransactionOperationsClient/customerCreditTransactionOperationsClientContext.js";
import {
  list as list_3,
  type ListOptions as ListOptions_3,
} from "./api/customerCreditTransactionOperationsClient/customerCreditTransactionOperationsClientOperations.js";
import {
  createCustomersClientContext,
  type CustomersClientContext,
  type CustomersClientOptions,
} from "./api/customersClientContext.js";
import {
  createCustomersOperationsClientContext,
  type CustomersOperationsClientContext,
  type CustomersOperationsClientOptions,
} from "./api/customersOperationsClient/customersOperationsClientContext.js";
import {
  create,
  type CreateOptions,
  delete_,
  type DeleteOptions,
  get,
  type GetOptions,
  list,
  type ListOptions,
  upsert,
  type UpsertOptions,
} from "./api/customersOperationsClient/customersOperationsClientOperations.js";
import type {
  CreateRequest_2,
  CreateRequest_3,
  CreateRequestNested,
  CustomerBillingStripeCreateCheckoutSessionRequest,
  CustomerBillingStripeCreateCustomerPortalSessionRequest,
  UpdateCreditGrantExternalSettlementRequest,
  UpsertRequest,
  UpsertRequest_2,
  UpsertRequest_3,
} from "./models/models.js";

export class CustomersClient {
  #context: CustomersClientContext
  customersOperationsClient: CustomersOperationsClient;
  customerBillingOperationsClient: CustomerBillingOperationsClient;
  customerCreditGrantsOperationsClient: CustomerCreditGrantsOperationsClient;
  customerCreditBalancesOperationsClient: CustomerCreditBalancesOperationsClient;
  customerCreditAdjustmentsOperationsClient: CustomerCreditAdjustmentsOperationsClient;
  customerCreditGrantExternalSettlementOperationsClient: CustomerCreditGrantExternalSettlementOperationsClient;
  customerCreditTransactionOperationsClient: CustomerCreditTransactionOperationsClient;
  customerChargesOperationsClient: CustomerChargesOperationsClient
  constructor(endpoint: string, options?: CustomersClientOptions) {
    this.#context = createCustomersClientContext(endpoint, options);
    this.customersOperationsClient = new CustomersOperationsClient(
      endpoint,
      options
    );;this
      .customerBillingOperationsClient = new CustomerBillingOperationsClient(
      endpoint,
      options
    );;this
      .customerCreditGrantsOperationsClient = new CustomerCreditGrantsOperationsClient(
      endpoint,
      options
    );;this
      .customerCreditBalancesOperationsClient = new CustomerCreditBalancesOperationsClient(
      endpoint,
      options
    );;this
      .customerCreditAdjustmentsOperationsClient = new CustomerCreditAdjustmentsOperationsClient(
      endpoint,
      options
    );;this
      .customerCreditGrantExternalSettlementOperationsClient = new CustomerCreditGrantExternalSettlementOperationsClient(
      endpoint,
      options
    );;this
      .customerCreditTransactionOperationsClient = new CustomerCreditTransactionOperationsClient(
      endpoint,
      options
    );;this
      .customerChargesOperationsClient = new CustomerChargesOperationsClient(
      endpoint,
      options
    );
  }
}
export class CustomerChargesOperationsClient {
  #context: CustomerChargesOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerChargesOperationsClientOptions,
  ) {
    this.#context = createCustomerChargesOperationsClientContext(
      endpoint,
      options
    );

  }
  list(customerId: string, options?: ListOptions_4) {
    return list_4(this.#context, customerId, options);
  }
}
export class CustomerCreditTransactionOperationsClient {
  #context: CustomerCreditTransactionOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerCreditTransactionOperationsClientOptions,
  ) {
    this.#context = createCustomerCreditTransactionOperationsClientContext(
      endpoint,
      options
    );

  }
  list(customerId: string, options?: ListOptions_3) {
    return list_3(this.#context, customerId, options);
  }
}
export class CustomerCreditGrantExternalSettlementOperationsClient {
  #context: CustomerCreditGrantExternalSettlementOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerCreditGrantExternalSettlementOperationsClientOptions,
  ) {
    this
      .#context = createCustomerCreditGrantExternalSettlementOperationsClientContext(
      endpoint,
      options
    );

  }
  async updateExternalSettlement(
    customerId: string,
    creditGrantId: string,
    body: UpdateCreditGrantExternalSettlementRequest,
    options?: UpdateExternalSettlementOptions,
  ) {
    return updateExternalSettlement(
      this.#context,
      customerId,
      creditGrantId,
      body,
      options
    );
  }
}
export class CustomerCreditAdjustmentsOperationsClient {
  #context: CustomerCreditAdjustmentsOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerCreditAdjustmentsOperationsClientOptions,
  ) {
    this.#context = createCustomerCreditAdjustmentsOperationsClientContext(
      endpoint,
      options
    );

  }
  async create(
    customerId: string,
    creditAdjustment: CreateRequest_3,
    options?: CreateOptions_3,
  ) {
    return create_3(this.#context, customerId, creditAdjustment, options);
  }
}
export class CustomerCreditBalancesOperationsClient {
  #context: CustomerCreditBalancesOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerCreditBalancesOperationsClientOptions,
  ) {
    this.#context = createCustomerCreditBalancesOperationsClientContext(
      endpoint,
      options
    );

  }
  async get(customerId: string, options?: GetOptions_4) {
    return get_4(this.#context, customerId, options);
  }
}
export class CustomerCreditGrantsOperationsClient {
  #context: CustomerCreditGrantsOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerCreditGrantsOperationsClientOptions,
  ) {
    this.#context = createCustomerCreditGrantsOperationsClientContext(
      endpoint,
      options
    );

  }
  async create(
    customerId: string,
    creditGrant: CreateRequestNested,
    options?: CreateOptions_2,
  ) {
    return create_2(this.#context, customerId, creditGrant, options);
  };
  async get(customerId: string, creditGrantId: string, options?: GetOptions_3) {
    return get_3(this.#context, customerId, creditGrantId, options);
  };
  list(customerId: string, options?: ListOptions_2) {
    return list_2(this.#context, customerId, options);
  }
}
export class CustomerBillingOperationsClient {
  #context: CustomerBillingOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerBillingOperationsClientOptions,
  ) {
    this.#context = createCustomerBillingOperationsClientContext(
      endpoint,
      options
    );

  }
  async get(customerId: string, options?: GetOptions_2) {
    return get_2(this.#context, customerId, options);
  };
  async upsert(
    customerId: string,
    body: UpsertRequest_2,
    options?: UpsertOptions_2,
  ) {
    return upsert_2(this.#context, customerId, body, options);
  };
  async upsertAppData(
    customerId: string,
    body: UpsertRequest_3,
    options?: UpsertAppDataOptions,
  ) {
    return upsertAppData(this.#context, customerId, body, options);
  };
  async createCheckoutSession(
    customerId: string,
    body: CustomerBillingStripeCreateCheckoutSessionRequest,
    options?: CreateCheckoutSessionOptions,
  ) {
    return createCheckoutSession(this.#context, customerId, body, options);
  };
  async createPortalSession(
    customerId: string,
    body: CustomerBillingStripeCreateCustomerPortalSessionRequest,
    options?: CreatePortalSessionOptions,
  ) {
    return createPortalSession(this.#context, customerId, body, options);
  }
}
export class CustomersOperationsClient {
  #context: CustomersOperationsClientContext
  constructor(endpoint: string, options?: CustomersOperationsClientOptions) {
    this.#context = createCustomersOperationsClientContext(endpoint, options);

  }
  async create(customer: CreateRequest_2, options?: CreateOptions) {
    return create(this.#context, customer, options);
  };
  async get(customerId: string, options?: GetOptions) {
    return get(this.#context, customerId, options);
  };
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async upsert(
    customerId: string,
    customer: UpsertRequest,
    options?: UpsertOptions,
  ) {
    return upsert(this.#context, customerId, customer, options);
  };
  async delete_(customerId: string, options?: DeleteOptions) {
    return delete_(this.#context, customerId, options);
  }
}
