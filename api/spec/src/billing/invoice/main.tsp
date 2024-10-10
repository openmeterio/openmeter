import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "../..";

// Package Contents
import "./bill.tsp";
import "./tax.tsp";
import "./cbc.tsp";
import "./l10n.tsp";
import "./org.tsp";
import "./currency.tsp";
import "./pay.tsp";
import "./om_cbc_notes.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace Invoice;
