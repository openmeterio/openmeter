import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "..";

// Package Contents
import "./entitlements.tsp";
import "./customer.tsp";
import "./grants.tsp";
import "./grant.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter;

// NOTE: Public types, operationIDs and friendlyNames of versioned packages should will use the version suffix.
