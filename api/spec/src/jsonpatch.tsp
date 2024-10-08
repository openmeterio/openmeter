import "@typespec/http";
import "./types.tsp";

namespace OpenMeter;

using TypeSpec.Http;

enum JSONPatchValueOperation {
  ADD: "add",
}

enum JSONPatchValuelessOperation {
  REMOVE: "remove",
}

model JSONPatchAdd<T, K extends string> {
  `op`: JSONPatchValueOperation;
  path: K;
  value: T;
}

model JSONPatchRemove<K extends string> {
  `op`: JSONPatchValuelessOperation;
  path: K;
}
