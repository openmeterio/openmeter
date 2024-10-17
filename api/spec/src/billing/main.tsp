import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "..";
import "./invoices/main.tsp";

// Package Contents
import "./profile.tsp";
import "./customeroverride.tsp";
import "./invoices.tsp";
import "./tax.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter.Billing;
