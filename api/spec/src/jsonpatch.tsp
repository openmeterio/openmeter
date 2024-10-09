import "@typespec/http";
import "./types.tsp";

namespace OpenMeter;

using TypeSpec.Http;

enum JSONPatchValueOperation {
  add: "add",
}

enum JSONPatchValuelessOperation {
  remove: "remove",
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
