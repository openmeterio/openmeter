import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "../..";
import "../../productcatalog";

// Package Contents
import "./invoice.tsp";
import "./tax.tsp";
import "./party.tsp";
import "./pay.tsp";
import "./docref.tsp";
import "./discounts.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter;
