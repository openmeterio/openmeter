/**
 * Emit per-operation request and response structs under `models/operations/`.
 *
 *   <op>Request: holds query/body parameters (queryParam-tagged for utils.PopulateQueryParams).
 *   <op>Response: holds HTTPMeta + optional Body for the success status.
 *
 * Path parameters are NOT included in the request struct — they're passed as
 * direct arguments to the method.
 *
 * v1 also defines the Option/Options types referenced by service methods.
 */
import type {
  Operation,
  QueryParam,
  SdkModel,
} from "../model/index.js";
import { formatType, optionalize, type FormatCtx } from "./format.js";
import { GENERATED_BANNER, Writer } from "./writer.js";

const OPS_PKG = "models/operations";

export function emitOperationsCommon(sdk: SdkModel): { path: string; content: string } {
  const w = new Writer();
  w.preamble(GENERATED_BANNER);
  w.packageName("operations");
  w.import("time");
  w.import(`${sdk.module}/retry`);

  w.line("// Options carries per-operation options.");
  w.line("type Options struct {");
  w.indent(() => {
    w.line("ServerURL  *string");
    w.line("Retries    *retry.Config");
    w.line("Timeout    *time.Duration");
    w.line("SetHeaders map[string]string");
  });
  w.line("}");
  w.blankLine();

  w.line("// Option configures Options.");
  w.line("type Option func(*Options, ...string) error");
  w.blankLine();

  w.line("const (");
  w.indent(() => {
    w.line(`SupportedOptionServerURL = "serverURL"`);
    w.line(`SupportedOptionRetries   = "retries"`);
    w.line(`SupportedOptionTimeout   = "timeout"`);
  });
  w.line(")");
  w.blankLine();

  // Provide concrete With* options at the operations level.
  w.line("// WithServerURL applies a per-request server URL override.");
  w.line("func WithServerURL(serverURL string) Option {");
  w.indent(() => {
    w.line("return func(o *Options, _ ...string) error {");
    w.indent(() => {
      w.line("o.ServerURL = &serverURL");
      w.line("return nil");
    });
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  w.line("// WithTimeout applies a per-request timeout.");
  w.line("func WithTimeout(timeout time.Duration) Option {");
  w.indent(() => {
    w.line("return func(o *Options, _ ...string) error {");
    w.indent(() => {
      w.line("o.Timeout = &timeout");
      w.line("return nil");
    });
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  w.line("// WithRetries applies a per-request retry config.");
  w.line("func WithRetries(config retry.Config) Option {");
  w.indent(() => {
    w.line("return func(o *Options, _ ...string) error {");
    w.indent(() => {
      w.line("o.Retries = &config");
      w.line("return nil");
    });
    w.line("}");
  });
  w.line("}");

  return { path: `${OPS_PKG}/options.go`, content: w.finish() };
}

export function emitOperationFiles(
  sdk: SdkModel,
  op: Operation,
  stringBackedTypes: ReadonlySet<string>,
): { path: string; content: string }[] {
  const out: { path: string; content: string }[] = [];
  const w = new Writer();
  w.preamble(GENERATED_BANNER);
  w.packageName("operations");
  const ctx: FormatCtx = { writer: w, module: sdk.module, currentPkg: OPS_PKG, stringBackedTypes };

  emitRequest(ctx, op);
  emitResponse(ctx, sdk, op);

  out.push({
    path: `${OPS_PKG}/${op.methodName.toLowerCase()}.go`,
    content: w.finish(),
  });
  return out;
}

function emitRequest(ctx: FormatCtx, op: Operation): void {
  const w = ctx.writer;
  const queryParams = op.params.filter((p): p is QueryParam => p.kind === "query");
  const hasFields = queryParams.length > 0 || op.body !== undefined;
  if (!hasFields) {
    // Still emit an empty struct so the method signature works.
    w.line(`type ${op.methodName}Request struct {}`);
    w.blankLine();
    return;
  }
  w.line(`type ${op.methodName}Request struct {`);
  w.indent(() => {
    for (const q of queryParams) {
      const fieldType = q.optional ? optionalize(q.type) : q.type;
      const typeStr = formatType(ctx, fieldType);
      const styleTag = q.style === "deepObject" ? `,style=deepObject` : "";
      const explodeTag = q.explode ? `,explode=true` : "";
      w.line(`${q.name} ${typeStr} \`queryParam:"name=${q.wireName}${styleTag}${explodeTag}"\``);
    }
    if (op.body) {
      const typeStr = formatType(ctx, op.body.type);
      w.line(`${op.body.name} ${typeStr} \`request:"mediaType=${op.body.contentType}"\``);
    }
  });
  w.line("}");
  w.blankLine();
}

function emitResponse(ctx: FormatCtx, sdk: SdkModel, op: Operation): void {
  const w = ctx.writer;
  w.import(`${sdk.module}/models/components`);
  // For v1, store all success bodies under a single `Body` field of `any`-typed
  // pointer or specific type if there's exactly one success response.
  const successResp = op.responses.find((r) => !r.isError && r.bodyType);
  w.line(`type ${op.methodName}Response struct {`);
  w.indent(() => {
    w.line("HTTPMeta components.HTTPMetadata");
    if (successResp && successResp.bodyType) {
      const typeStr = formatType(ctx, successResp.bodyType);
      w.line(`Body *${typeStr}`);
    }
  });
  w.line("}");
}
