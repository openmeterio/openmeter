export type { SdkModel, Server, ServerVariable } from "./sdk.js";
export type { Service } from "./service.js";
export type { Operation, OperationBody, OperationResponse } from "./operation.js";
export type { Param, PathParam, QueryParam, HeaderParam } from "./parameter.js";
export type {
  Decl,
  StructDecl,
  StructField,
  EnumDecl,
  EnumMember,
  DiscriminatedUnionDecl,
  DiscriminatedVariant,
  HeuristicUnionDecl,
  HeuristicVariant,
  AliasDecl,
  DiscriminatorVariantInfo,
} from "./decl.js";
export type {
  GoType,
  GoScalar,
  GoSlice,
  GoMap,
  GoPointer,
  GoNamed,
  GoTime,
  GoAny,
} from "./type.js";
export {
  goString,
  goBool,
  goInt32,
  goInt64,
  goFloat32,
  goFloat64,
  goBytes,
  goTime,
  goAny,
  ptr,
  slice,
  mapOf,
  named,
} from "./type.js";
