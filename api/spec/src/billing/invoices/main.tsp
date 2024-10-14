import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "../..";

// Package Contents
import "./invoice.tsp";
import "./tax.tsp";
import "./codes.tsp";
import "./org.tsp";
import "./pay.tsp";
import "./product.tsp";
import "./docref.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace Invoice;
