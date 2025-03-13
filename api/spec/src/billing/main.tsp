import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "..";
import "./invoices/main.tsp";

// Package Contents
import "./profile.tsp";
import "./customeroverride.tsp";
import "./invoices.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;
