import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "..";

// Package Contents
import "./entitlements.tsp";
import "./feature.tsp";
import "./grant.tsp";
import "./management.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter.Entitlements;
