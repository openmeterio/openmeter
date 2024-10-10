import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "..";
import "./invoice/main.tsp";

// Package Contents
import "./types.tsp";
import "./profile.tsp";
import "./customer.tsp";
import "./invoices.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter.Billing;
