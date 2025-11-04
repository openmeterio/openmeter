import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "..";

// Package Contents
import "./entitlements.tsp";
import "../productcatalog/features.tsp";
import "./grant.tsp";
import "./subjects.tsp";
import "./customer.tsp";

namespace OpenMeter;
