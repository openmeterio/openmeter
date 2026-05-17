/**
 * Build the intermediate SdkModel from a TypeSpec Program.
 *
 * The builder is intentionally permissive: when a TypeSpec construct doesn't
 * map cleanly to Go, it falls back to `any` rather than throwing. The compile
 * step (go vet) will catch real problems downstream.
 */
import {
  type EmitContext,
  type Enum,
  type Interface,
  type Model,
  type ModelProperty,
  type Namespace,
  type Program,
  type Scalar,
  type Type,
  type Union,
  type UnionVariant,
  getDiscriminatedUnion,
  getDiscriminator,
  getDoc,
  getFriendlyName,
  getSummary,
  isErrorModel,
  isTemplateDeclaration,
  isTemplateInstance,
  listServices,
  navigateProgram,
} from "@typespec/compiler";
import {
  type HttpOperation,
  type HttpOperationParameter,
  type HttpOperationResponse,
  type HttpServer,
  getAllHttpServices,
  getServers,
  isBody,
  isBodyRoot,
  isHeader,
  isPathParam,
  isQueryParam,
  isStatusCode,
} from "@typespec/http";
import { getOperationId } from "@typespec/openapi";

import type { GoEmitterOptions } from "../lib.js";
import {
  type AliasDecl,
  type Decl,
  type DiscriminatedUnionDecl,
  type DiscriminatedVariant,
  type EnumDecl,
  type EnumMember,
  type HeuristicUnionDecl,
  type HeuristicVariant,
  type Operation,
  type OperationBody,
  type OperationResponse,
  type Param,
  type SdkModel,
  type Server,
  type Service,
  type StructDecl,
  type StructField,
  goAny,
  goBool,
  goBytes,
  goFloat32,
  goFloat64,
  goInt32,
  goInt64,
  goString,
  goTime,
  type GoType,
  mapOf,
  named,
  slice,
} from "./index.js";
import { fileBase, pascal } from "./naming.js";

const COMPONENTS_PKG = "models/components";

interface BuildCtx {
  readonly program: Program;
  readonly components: Decl[];
  /** Decls already emitted, keyed by canonical Go name. */
  readonly seen: Map<string, Decl>;
  /** TypeSpec Type identity -> Go name, so the same Type always maps to the same Decl. */
  readonly typeToName: Map<Type, string>;
  /**
   * Maps raw TypeSpec type names (e.g. "AppStripe") to the canonical Go name
   * we picked for them (e.g. "BillingAppStripe"). Lets a second Type instance
   * arriving with the same raw name (typically a synthetic copy in a union
   * variant context, lacking the @friendlyName decorator) collapse to the
   * already-emitted Decl.
   */
  readonly rawNameToGoName: Map<string, string>;
  /** Types currently being built, to break cycles. */
  readonly inProgress: Set<Type>;
  /** Status codes used by any operation response that is an error. */
  readonly errorStatusCodes: Set<number>;
}

export function buildSdkModel(
  context: EmitContext<GoEmitterOptions>,
): SdkModel {
  const program = context.program;
  const module = context.options.module;
  const packageName = context.options["package-name"] ?? deriveDefaultPackageName(module);
  const sdkVersion = context.options["sdk-version"] ?? "0.0.1";
  const userAgent = context.options["user-agent"] ?? `${packageName}-go/${sdkVersion}`;

  const ctx: BuildCtx = {
    program,
    components: [],
    seen: new Map(),
    typeToName: new Map(),
    rawNameToGoName: new Map(),
    inProgress: new Set(),
    errorStatusCodes: new Set(),
  };

  // 1. Walk all reachable type declarations so we emit components for everything,
  //    not only types referenced from operations. This matches Speakeasy's behavior
  //    (it emits every named type in the spec, not just operation-reachable ones).
  navigateProgram(
    program,
    {
      model(m) {
        if (shouldSkipDecl(m)) return;
        emitDecl(ctx, m);
      },
      enum(e) {
        if (shouldSkipDecl(e)) return;
        emitDecl(ctx, e);
      },
      union(u) {
        if (shouldSkipDecl(u)) return;
        emitDecl(ctx, u);
      },
      scalar(s) {
        if (shouldSkipDecl(s)) return;
        emitDecl(ctx, s);
      },
    },
    { includeTemplateDeclaration: false },
  );

  // 2. Walk HTTP services for operations.
  const [services, diagnostics] = getAllHttpServices(program);
  void diagnostics; // structural diagnostics handled by the compiler itself

  // Service buckets keyed by serviceName; operation IDs are deduped so the
  // same operation reachable from multiple namespaces only appears once.
  const byService = new Map<string, { ops: Operation[]; seen: Set<string> }>();

  for (const httpService of services) {
    for (const op of httpService.operations) {
      const serviceName = deriveServiceName(op);
      const goOp = buildOperation(ctx, op, serviceName);
      if (!goOp) continue;
      let bucket = byService.get(serviceName);
      if (!bucket) {
        bucket = { ops: [], seen: new Set() };
        byService.set(serviceName, bucket);
      }
      if (bucket.seen.has(goOp.id)) continue;
      bucket.seen.add(goOp.id);
      bucket.ops.push(goOp);
    }
  }

  const goServices: Service[] = [];
  for (const [serviceName, bucket] of byService) {
    const structName = `OpenMeter${serviceName}`;
    goServices.push({
      name: serviceName,
      fileName: `${fileBase(structName)}.go`,
      rootFieldName: structName,
      structName,
      ctorName: `new${structName}`,
      operations: bucket.ops,
    });
  }

  // 3. Servers.
  const serverList: Server[] = [];
  const globalNs = program.getGlobalNamespaceType();
  const declaredServers = collectServers(program, globalNs);
  for (const s of declaredServers) {
    serverList.push({
      url: s.url,
      description: s.description ?? "",
      variables: Array.from(s.parameters.entries()).map(([name, prop]) => ({
        name,
        default: extractDefault(prop),
      })),
    });
  }

  // Service title + summary from @service / @summary.
  const tspServices = listServices(program);
  const primaryService = tspServices[0];
  const title = primaryService?.title ?? packageName;
  const summary = primaryService?.type
    ? getSummary(program, primaryService.type) ?? getDoc(program, primaryService.type)
    : undefined;

  return {
    module,
    packageName,
    sdkVersion,
    userAgent,
    title,
    summary,
    servers: serverList,
    services: goServices,
    components: ctx.components,
    errorStatusCodes: [...ctx.errorStatusCodes].sort((a, b) => a - b),
  };
}

// --------------------------------------------------------------------------
// Decl emission
// --------------------------------------------------------------------------

function shouldSkipDecl(t: Type): boolean {
  // Skip stdlib types (Record, Array, etc.) and template declarations.
  if ("namespace" in t && t.namespace) {
    const fullName = getNamespaceFullName(t.namespace);
    if (
      fullName === "TypeSpec" ||
      fullName.startsWith("TypeSpec.") ||
      fullName === "Reflection" ||
      fullName === ""
    ) {
      // Allow user-defined types in the global namespace; the global namespace
      // returns "" from getNamespaceFullName but `t.namespace` will be set to
      // it. Still skip if it has no name.
      if (fullName.startsWith("TypeSpec") || fullName === "Reflection") return true;
    }
  }
  if ("name" in t && (!t.name || typeof t.name !== "string")) return true;
  if ("name" in t && typeof t.name === "string" && t.name === "") return true;
  if (t.kind === "Model" || t.kind === "Union" || t.kind === "Enum" || t.kind === "Scalar") {
    if (isTemplateDeclaration(t as any)) return true;
  }
  return false;
}

function emitDecl(ctx: BuildCtx, t: Type): GoType | undefined {
  if (!("name" in t) || typeof t.name !== "string" || !t.name) return undefined;
  // Same Type seen before? Return its name, never emit twice.
  const prior = ctx.typeToName.get(t);
  if (prior) return named(prior, COMPONENTS_PKG);

  const rawName = t.name;
  // If a sibling Type instance with the same raw TypeSpec name already
  // resolved to a Go name, reuse it. This collapses synthetic copies
  // (e.g. union variant instances that lack @friendlyName) onto the canonical
  // Decl picked the first time we saw the name.
  const aliased = ctx.rawNameToGoName.get(rawName);
  if (aliased) {
    ctx.typeToName.set(t, aliased);
    return named(aliased, COMPONENTS_PKG);
  }

  // Prefer @friendlyName for template instances (e.g. Shared.PagePaginatedResponse<Meter>
  // -> MeterPagePaginatedResponse). Falls back to the raw type name otherwise.
  const friendly = getFriendlyName(ctx.program, t as Model | Enum | Union | Scalar);
  const finalName =
    friendly && typeof friendly === "string" && friendly.trim() ? friendly : rawName;
  const name = pascal(finalName);
  ctx.typeToName.set(t, name);
  ctx.rawNameToGoName.set(rawName, name);
  if (ctx.seen.has(name)) return named(name, COMPONENTS_PKG);
  if (ctx.inProgress.has(t)) return named(name, COMPONENTS_PKG);
  ctx.inProgress.add(t);
  try {
    let decl: Decl | undefined;
    if (t.kind === "Model") {
      decl = buildStructDecl(ctx, t, name);
    } else if (t.kind === "Enum") {
      decl = buildEnumDecl(ctx, t, name);
    } else if (t.kind === "Union") {
      decl = buildUnionDecl(ctx, t, name);
    } else if (t.kind === "Scalar") {
      decl = buildAliasDecl(ctx, t, name);
    }
    if (decl) {
      ctx.seen.set(name, decl);
      ctx.components.push(decl);
    }
    return named(name, COMPONENTS_PKG);
  } finally {
    ctx.inProgress.delete(t);
  }
}

function buildStructDecl(ctx: BuildCtx, m: Model, name: string): StructDecl {
  const fields: StructField[] = [];
  for (const [propName, prop] of m.properties) {
    if (isHttpEnvelopeProperty(ctx.program, prop)) continue;
    if (!propName || propName.startsWith("_")) continue;
    const fieldType = resolveType(ctx, prop.type);
    fields.push({
      name: pascal(propName),
      jsonName: propName,
      type: fieldType,
      optional: prop.optional === true,
      doc: getDoc(ctx.program, prop),
    });
  }
  return {
    kind: "struct",
    name,
    doc: getDoc(ctx.program, m),
    fields,
  };
}

/**
 * Properties that are part of the HTTP envelope (status code, body marker,
 * header, path/query annotations) are metadata, not data. Filter them out
 * when emitting a struct intended as a data type.
 */
function isHttpEnvelopeProperty(program: Program, prop: ModelProperty): boolean {
  if (isStatusCode(program, prop)) return true;
  if (isHeader(program, prop)) return true;
  if (isPathParam(program, prop)) return true;
  if (isQueryParam(program, prop)) return true;
  if (isBody(program, prop)) return true;
  if (isBodyRoot(program, prop)) return true;
  return false;
}

function buildEnumDecl(ctx: BuildCtx, e: Enum, name: string): EnumDecl {
  const members: EnumMember[] = [];
  for (const [memberName, member] of e.members) {
    members.push({
      name: pascal(memberName),
      value: String(member.value ?? memberName),
    });
  }
  return {
    kind: "enum",
    name,
    doc: getDoc(ctx.program, e),
    members,
    forwardCompatible: true,
  };
}

function buildUnionDecl(
  ctx: BuildCtx,
  u: Union,
  name: string,
): EnumDecl | DiscriminatedUnionDecl | HeuristicUnionDecl {
  const variants = Array.from(u.variants.values());

  // A) All variants are string literals => emit as an Enum.
  if (variants.every((v) => isStringLiteral(v.type))) {
    return {
      kind: "enum",
      name,
      doc: getDoc(ctx.program, u),
      members: variants.map((v) => ({
        name: pascal(String(v.name)),
        value: stringLiteralValue(v.type),
      })),
      forwardCompatible: true,
    };
  }

  // B) Discriminator detection.
  const disc = getUnionDiscriminator(ctx.program, u);
  if (disc) {
    let discriminatorEnumName: string | undefined;
    const dvariants: DiscriminatedVariant[] = [];
    for (const v of variants) {
      const variantType = v.type;
      if (variantType.kind !== "Model" || !("name" in variantType) || !variantType.name) continue;
      const variantRef = emitDecl(ctx, variantType);
      if (!variantRef || variantRef.kind !== "named") continue;
      const { value, enumTypeName } = resolveDiscriminatorValueAndType(ctx, variantType, disc, v.name);
      if (enumTypeName && !discriminatorEnumName) discriminatorEnumName = enumTypeName;
      const variantValue = value ?? String(v.name);
      dvariants.push({
        name: pascal(variantRef.name),
        value: variantValue,
        typeRef: { name: variantRef.name, pkg: variantRef.pkg },
      });
      // Mutate the previously-emitted variant Struct to mark it as a
      // discriminator variant — add the per-variant singleton enum + Type field.
      const existing = ctx.seen.get(variantRef.name);
      if (existing && existing.kind === "struct" && !existing.discriminatorVariant) {
        // The Decl is readonly at the type level, but we built it ourselves
        // moments ago; mutating in-place avoids a multi-pass design.
        (existing as { discriminatorVariant?: unknown }).discriminatorVariant = {
          unionName: name,
          value: variantValue,
          singletonEnumName: `Type${pascalSnake(variantValue)}`,
        };
      }
    }
    return {
      kind: "discriminated-union",
      name,
      doc: getDoc(ctx.program, u),
      discriminatorProperty: disc,
      discriminatorEnumName,
      variants: dvariants,
    };
  }

  // C) Mixed-kind union => heuristic union.
  const hvariants: HeuristicVariant[] = [];
  for (const v of variants) {
    const vType = resolveType(ctx, v.type);
    const vName = variantNameFor(v, vType);
    hvariants.push({ name: vName, type: vType });
  }
  return {
    kind: "heuristic-union",
    name,
    doc: getDoc(ctx.program, u),
    variants: hvariants,
  };
}

function pascalSnake(s: string): string {
  return s
    .split(/[_\-]/)
    .filter(Boolean)
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1).toLowerCase())
    .join("");
}

function buildAliasDecl(ctx: BuildCtx, s: Scalar, name: string): AliasDecl | undefined {
  const target = resolveScalar(ctx, s);
  // Don't emit aliases for plain primitive scalars (they map to Go primitives directly).
  // Only emit named aliases for user-defined scalars in user namespaces.
  const ns = s.namespace ? getNamespaceFullName(s.namespace) : "";
  if (ns === "TypeSpec" || ns.startsWith("TypeSpec.")) return undefined;
  return {
    kind: "alias",
    name,
    doc: getDoc(ctx.program, s),
    target,
  };
}

// --------------------------------------------------------------------------
// Type resolution
// --------------------------------------------------------------------------

function resolveType(ctx: BuildCtx, t: Type): GoType {
  switch (t.kind) {
    case "Scalar":
      return resolveScalar(ctx, t);
    case "Model":
      return resolveModel(ctx, t);
    case "Enum": {
      const ref = emitDecl(ctx, t);
      return ref ?? goString;
    }
    case "Union": {
      const ref = emitDecl(ctx, t);
      return ref ?? goAny;
    }
    case "Intrinsic": {
      const name = t.name;
      if (name === "null" || name === "void" || name === "never") return goAny;
      if (name === "unknown") return goAny;
      return goAny;
    }
    case "String":
      return goString;
    case "Number":
      return goFloat64;
    case "Boolean":
      return goBool;
    default:
      return goAny;
  }
}

function resolveScalar(_ctx: BuildCtx, s: Scalar): GoType {
  const baseKind = getScalarBaseKind(s);
  switch (baseKind) {
    case "boolean":
      return goBool;
    case "string":
    case "url":
      return goString;
    case "bytes":
      return goBytes;
    case "int8":
    case "int16":
    case "int32":
    case "uint8":
    case "uint16":
    case "uint32":
      return goInt32;
    case "int64":
    case "safeint":
    case "integer":
    case "numeric":
    case "uint64":
      return goInt64;
    case "float32":
      return goFloat32;
    case "float":
    case "float64":
    case "decimal":
    case "decimal128":
      return goFloat64;
    case "utcDateTime":
    case "offsetDateTime":
      return goTime;
    case "plainDate":
    case "plainTime":
    case "duration":
      return goString;
    default:
      return goString;
  }
}

function resolveModel(ctx: BuildCtx, m: Model): GoType {
  // Array
  if (m.name === "Array" && m.indexer) {
    return slice(resolveType(ctx, m.indexer.value));
  }
  // Record
  if (m.name === "Record" && m.indexer) {
    return mapOf(resolveType(ctx, m.indexer.value));
  }
  // Anonymous model (inline object) => represent as `map[string]any` for v1.
  // The reference SDK promotes these into named types; in v1 we accept the loss.
  if (!m.name) return mapOf(goAny);

  // Named model => emit and return reference.
  const ref = emitDecl(ctx, m);
  return ref ?? mapOf(goAny);
}

function getScalarBaseKind(s: Scalar): string {
  // Walk up the baseScalar chain until we hit a TypeSpec built-in.
  let cur: Scalar | undefined = s;
  while (cur) {
    const ns = cur.namespace ? getNamespaceFullName(cur.namespace) : "";
    if (ns === "TypeSpec") return cur.name;
    cur = cur.baseScalar;
  }
  return s.name;
}

// --------------------------------------------------------------------------
// Operation building
// --------------------------------------------------------------------------

function buildOperation(
  ctx: BuildCtx,
  op: HttpOperation,
  serviceName: string,
): Operation | undefined {
  const opIdAnnotated = getOperationId(ctx.program, op.operation);
  const opId = opIdAnnotated ?? op.operation.name;
  const methodName = pascal(opId);

  const params: Param[] = [];
  for (const hp of op.parameters.parameters) {
    const p = buildParameter(ctx, hp);
    if (p) params.push(p);
  }

  let body: OperationBody | undefined;
  if (op.parameters.body && op.parameters.body.type) {
    body = {
      name: pascal(getBodyPropertyName(op.parameters.body)),
      type: resolveType(ctx, op.parameters.body.type),
      contentType: op.parameters.body.contentTypes[0] ?? "application/json",
    };
  }

  const responses: OperationResponse[] = [];
  for (const r of op.responses) {
    const built = buildResponse(ctx, r);
    responses.push(...built);
  }

  return {
    id: opId,
    methodName,
    service: serviceName,
    verb: op.verb.toUpperCase() as Operation["verb"],
    path: op.path,
    params,
    body,
    responses,
    doc: getDoc(ctx.program, op.operation),
  };
}

function buildParameter(ctx: BuildCtx, hp: HttpOperationParameter): Param | undefined {
  const prop = hp.param;
  const wireName = "name" in hp && typeof hp.name === "string" ? hp.name : prop.name;
  const baseType = resolveType(ctx, prop.type);
  const optional = prop.optional === true;
  const doc = getDoc(ctx.program, prop);
  const fieldName = pascal(prop.name);
  if (hp.type === "path") {
    return {
      kind: "path",
      name: fieldName,
      wireName,
      type: baseType,
      doc,
    };
  }
  if (hp.type === "query") {
    const style = isDeepObject(hp) ? "deepObject" : "form";
    return {
      kind: "query",
      name: fieldName,
      wireName,
      type: baseType,
      optional,
      style,
      explode: hp.explode === true,
      doc,
    };
  }
  if (hp.type === "header") {
    return {
      kind: "header",
      name: fieldName,
      wireName,
      type: baseType,
      optional,
      doc,
    };
  }
  // Cookie params unsupported in v1.
  return undefined;
}

function buildResponse(
  ctx: BuildCtx,
  r: HttpOperationResponse,
): OperationResponse[] {
  const out: OperationResponse[] = [];
  const statuses = expandStatusCodes(r.statusCodes);
  const declaredError =
    r.type && r.type.kind === "Model" && isErrorModel(ctx.program, r.type);
  // Treat any 4xx/5xx response as an error for purposes of emitting an
  // apierrors type. Real-world specs (OpenMeter's included) rarely apply
  // @error to error-shaped models; @statusCode on the model is enough.
  const isErr = declaredError || statuses.some((s) => s >= 400);
  for (const status of statuses) {
    if (isErr) ctx.errorStatusCodes.add(status);
    let bodyType: GoType | undefined;
    let contentType: string | undefined;
    if (r.responses.length > 0 && r.responses[0]!.body && r.responses[0]!.body!.type) {
      bodyType = resolveType(ctx, r.responses[0]!.body!.type);
      contentType = r.responses[0]!.body!.contentTypes[0];
    }
    out.push({
      status,
      bodyType,
      contentType,
      isError: !!isErr,
      errorTypeName: isErr ? errorTypeNameForStatus(status) : undefined,
    });
  }
  return out;
}

function expandStatusCodes(s: HttpOperationResponse["statusCodes"]): number[] {
  if (typeof s === "number") return [s];
  if (typeof s === "string") {
    // wildcard "*" => map to a generic 500 catch-all; this is rough but matches
    // Speakeasy's default-response handling.
    return [500];
  }
  // Range
  const out: number[] = [];
  for (let i = s.start; i <= s.end; i++) out.push(i);
  return out;
}

function errorTypeNameForStatus(status: number): string {
  switch (status) {
    case 400:
      return "BadRequestError";
    case 401:
      return "UnauthorizedError";
    case 403:
      return "ForbiddenError";
    case 404:
      return "NotFoundError";
    case 409:
      return "ConflictError";
    case 410:
      return "GoneError";
    case 413:
      return "PayloadTooLargeError";
    case 415:
      return "UnsupportedMediaTypeError";
    case 422:
      return "UnprocessableContentError";
    case 429:
      return "TooManyRequestsError";
    case 500:
      return "InternalError";
    case 501:
      return "NotImplementedError";
    case 503:
      return "ServiceUnavailableError";
    default:
      return `APIError${status}`;
  }
}

// --------------------------------------------------------------------------
// Service naming
// --------------------------------------------------------------------------

function deriveServiceName(op: HttpOperation): string {
  // Prefer the interface name (matches "Customers" in `interface CustomersOperations`).
  const container = op.container;
  if (container.kind === "Interface") {
    return pascal(stripOperationsSuffix(container.name));
  }
  if (container.kind === "Namespace") {
    return pascal(container.name);
  }
  return "Default";
}

function stripOperationsSuffix(name: string): string {
  if (name.endsWith("Endpoints")) return name.slice(0, -"Endpoints".length);
  if (name.endsWith("Operations")) return name.slice(0, -"Operations".length);
  return name;
}

// --------------------------------------------------------------------------
// Server collection
// --------------------------------------------------------------------------

function collectServers(program: Program, ns: Namespace): readonly HttpServer[] {
  const found: HttpServer[] = [];
  const direct = getServers(program, ns);
  if (direct) found.push(...direct);
  for (const child of ns.namespaces.values()) {
    found.push(...collectServers(program, child));
  }
  return found;
}

function extractDefault(prop: ModelProperty): string | undefined {
  // Default values come through prop.defaultValue in newer TypeSpec versions.
  const dv = (prop as ModelProperty & { defaultValue?: { value?: unknown } }).defaultValue;
  if (dv && typeof dv.value === "string") return dv.value;
  return undefined;
}

// --------------------------------------------------------------------------
// Misc helpers
// --------------------------------------------------------------------------

function getNamespaceFullName(ns: Namespace): string {
  const parts: string[] = [];
  let cur: Namespace | undefined = ns;
  while (cur && cur.name) {
    parts.unshift(cur.name);
    cur = cur.namespace;
  }
  return parts.join(".");
}

function isStringLiteral(t: Type): boolean {
  return t.kind === "String";
}

function stringLiteralValue(t: Type): string {
  if (t.kind === "String") return t.value;
  return "";
}

function getUnionDiscriminator(program: Program, u: Union): string | undefined {
  // `@discriminated` decorator (modern TypeSpec).
  const [discUnion] = getDiscriminatedUnion(program, u);
  if (discUnion) {
    return discUnion.options.discriminatorPropertyName ?? "kind";
  }
  // Legacy `@discriminator` decorator.
  const d = getDiscriminator(program, u);
  if (d) return d.propertyName;
  return undefined;
}

/**
 * Resolve a discriminator value for a variant model.
 *
 * Returns:
 *   value: the JSON literal of the discriminator (e.g. "stripe").
 *   enumTypeName: if the discriminator property is typed as a TypeSpec enum,
 *     the Go name of that enum (e.g. "AppType"). The emitter uses this to
 *     reference an existing enum instead of synthesizing a parallel parent enum.
 *
 * Handles three shapes:
 *   1. The variant model declares the discriminator directly with a string literal:
 *      `model X { type: "x" }`           -> value="x", enumTypeName=undefined
 *   2. The variant inherits from a templated base with a constrained type parameter
 *      that resolves to a single enum member:
 *      `model X is Base<MyEnum.X>`       -> value="x", enumTypeName="MyEnum"
 *   3. The variant is itself instantiated via a templated parent — in which case
 *      we fall through and the caller supplies the union variant key as fallback.
 */
function resolveDiscriminatorValueAndType(
  ctx: BuildCtx,
  m: Model,
  discProp: string,
  variantKey: string | symbol,
): { value: string | undefined; enumTypeName: string | undefined } {
  void variantKey;
  const prop = m.properties.get(discProp);
  if (!prop) return { value: undefined, enumTypeName: undefined };
  const t = prop.type;
  if (t.kind === "String") {
    return { value: t.value, enumTypeName: undefined };
  }
  if (t.kind === "EnumMember") {
    // Reference to a single enum member like `AppType.Stripe`.
    const memberValue = typeof t.value === "string" ? t.value : t.name;
    const parentEnum = t.enum;
    if (parentEnum && typeof parentEnum.name === "string") {
      // Emit the enum decl (returns its canonical Go name) so we reference the
      // actually-emitted name — accounting for @friendlyName rewrites.
      const ref = emitDecl(ctx, parentEnum);
      const goName =
        ref && ref.kind === "named"
          ? ref.name
          : ctx.rawNameToGoName.get(parentEnum.name) ?? pascal(parentEnum.name);
      return { value: String(memberValue), enumTypeName: goName };
    }
    return { value: String(memberValue), enumTypeName: undefined };
  }
  if (t.kind === "Enum") {
    // Whole enum reference — emit and use its canonical Go name.
    const ref = emitDecl(ctx, t);
    const goName =
      ref && ref.kind === "named"
        ? ref.name
        : ctx.rawNameToGoName.get(t.name) ?? pascal(t.name);
    return { value: undefined, enumTypeName: goName };
  }
  return { value: undefined, enumTypeName: undefined };
}

function variantNameFor(v: UnionVariant, vType: GoType): string {
  if (typeof v.name === "string" && v.name) return pascal(v.name);
  if (vType.kind === "named") return vType.name;
  if (vType.kind === "scalar") return pascal(vType.name);
  if (vType.kind === "slice") return "List";
  return "Value";
}

function isDeepObject(hp: HttpOperationParameter): boolean {
  if (hp.type !== "query") return false;
  // HttpOperationQueryParameter includes a `format`-style field on newer typespec,
  // but the canonical check is whether `style` is "deepObject" — we tolerate both
  // shapes.
  const anyHp = hp as HttpOperationParameter & { format?: string; style?: string };
  if (anyHp.style === "deepObject") return true;
  if (anyHp.format === "deepObject") return true;
  return false;
}

function getBodyPropertyName(body: HttpOperation["parameters"]["body"]): string {
  if (!body) return "Body";
  // If the body is sourced from a named property (not `_`), use its name.
  const props = (body as { property?: ModelProperty }).property;
  if (props && props.name && props.name !== "_") return props.name;
  // Otherwise fall back to the body type's name (e.g. "GovernanceQueryRequest").
  if (body.type && "name" in body.type && typeof body.type.name === "string" && body.type.name) {
    return body.type.name;
  }
  return "body";
}

function deriveDefaultPackageName(module: string): string {
  const segments = module.split("/").filter(Boolean);
  const last = segments[segments.length - 1] ?? "sdk";
  // Strip trailing /vN versioning (e.g. ".../foo/v2" -> "foo").
  if (/^v\d+$/.test(last) && segments.length >= 2) {
    return sanitizePkg(segments[segments.length - 2]!);
  }
  return sanitizePkg(last);
}

function sanitizePkg(name: string): string {
  // Go package names: lowercase, no hyphens.
  return name.replace(/[^a-zA-Z0-9_]/g, "").toLowerCase() || "sdk";
}

// Re-export — for callers that don't want to pull from "@typespec/compiler".
export { isTemplateInstance };
export type { Interface };
