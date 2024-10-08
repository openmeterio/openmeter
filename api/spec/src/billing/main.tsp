import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "..";

// Package Contents
import "./types.tsp";
import "./profile.tsp";
import "./customer.tsp";
import "./pending.tsp";
import "./invoice.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter.Billing;
