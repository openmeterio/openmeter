/**
 * Emit `openmeter<Service>.go` — one file per service.
 *
 * Each method is a self-contained HTTP call:
 *   - Build URL from base + path template + path params
 *   - Marshal body if present (application/json)
 *   - Apply query params via utils.PopulateQueryParams
 *   - Apply timeout if configured
 *   - Send request
 *   - Decode response by status code
 *
 * v1: NO retry, NO hooks. The SDKConfiguration has the wiring; methods skip
 * actually calling them. This keeps the operation method bodies tractable
 * while still producing a compilable, exercise-able SDK.
 */
import type {
  Operation,
  OperationResponse,
  PathParam,
  QueryParam,
  SdkModel,
  Service,
} from "../model/index.js";
import { formatType, type FormatCtx } from "./format.js";
import { GENERATED_BANNER, Writer } from "./writer.js";

export function emitServiceFile(
  sdk: SdkModel,
  service: Service,
  stringBackedTypes: ReadonlySet<string>,
): { path: string; content: string } {
  const w = new Writer();
  w.preamble(GENERATED_BANNER);
  w.packageName(sdk.packageName);

  // Common imports (we always need these)
  w.import(`${sdk.module}/internal/config`);
  w.import(`${sdk.module}/internal/hooks`);
  w.import(`${sdk.module}/internal/utils`);
  w.import(`${sdk.module}/models/operations`);
  w.import("context");
  w.import("fmt");
  w.import("net/http");
  w.import("net/url");

  // Struct + constructor
  w.line(`type ${service.structName} struct {`);
  w.indent(() => {
    w.line("rootSDK          *OpenMeter");
    w.line("sdkConfiguration config.SDKConfiguration");
    w.line("hooks            *hooks.Hooks");
  });
  w.line("}");
  w.blankLine();

  w.line(`func ${service.ctorName}(rootSDK *OpenMeter, sdkConfig config.SDKConfiguration, h *hooks.Hooks) *${service.structName} {`);
  w.indent(() => {
    w.line(`return &${service.structName}{`);
    w.indent(() => {
      w.line("rootSDK:          rootSDK,");
      w.line("sdkConfiguration: sdkConfig,");
      w.line("hooks:            h,");
    });
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // Methods
  const ctx: FormatCtx = { writer: w, module: sdk.module, currentPkg: "", stringBackedTypes };
  for (const op of service.operations) {
    emitMethod(ctx, service, op);
    w.blankLine();
  }

  return { path: service.fileName, content: w.finish() };
}

function emitMethod(ctx: FormatCtx, service: Service, op: Operation): void {
  const w = ctx.writer;
  const requestStructName = `${op.methodName}Request`;
  const responseStructName = `${op.methodName}Response`;

  // Method signature
  if (op.doc) w.doc(op.doc);
  const pathParams = op.params.filter((p): p is PathParam => p.kind === "path");
  const queryParams = op.params.filter((p): p is QueryParam => p.kind === "query");

  const sigArgs: string[] = ["ctx context.Context"];
  for (const pp of pathParams) {
    sigArgs.push(`${camel(pp.name)} ${formatType(ctx, pp.type)}`);
  }
  if (op.body) {
    sigArgs.push(`request operations.${requestStructName}`);
  } else if (queryParams.length > 0) {
    sigArgs.push(`request operations.${requestStructName}`);
  }
  sigArgs.push("opts ...operations.Option");

  w.line(
    `func (s *${service.structName}) ${op.methodName}(${sigArgs.join(", ")}) (*operations.${responseStructName}, error) {`,
  );
  w.indent(() => {
    // Apply options (best-effort; v1 supports timeout only via SDKConfiguration)
    w.line("o := operations.Options{}");
    w.line("supportedOptions := []string{}");
    w.line("for _, opt := range opts {");
    w.indent(() => {
      w.line("if err := opt(&o, supportedOptions...); err != nil {");
      w.indent(() => w.line(`return nil, fmt.Errorf("error applying option: %w", err)`));
      w.line("}");
    });
    w.line("}");
    w.line("_ = o");
    w.blankLine();

    // Base URL
    w.line("var baseURL string");
    w.line("if o.ServerURL == nil {");
    w.indent(() => emitBaseURL(w));
    w.line("} else {");
    w.indent(() => w.line("baseURL = *o.ServerURL"));
    w.line("}");
    w.blankLine();

    // Path: substitute path params.
    if (pathParams.length === 0) {
      w.line(`opURL, err := url.JoinPath(baseURL, ${JSON.stringify(op.path)})`);
    } else {
      // Build path: replace {name} with fmt.Sprintf("%v", name)
      let pathExpr = JSON.stringify(op.path);
      for (const pp of pathParams) {
        const local = camel(pp.name);
        pathExpr = pathExpr.replace(
          `{${pp.wireName}}`,
          `" + fmt.Sprintf("%v", ${local}) + "`,
        );
      }
      w.line(`opURL, err := url.JoinPath(baseURL, ${pathExpr})`);
    }
    w.line("if err != nil {");
    w.indent(() => w.line(`return nil, fmt.Errorf("error generating URL: %w", err)`));
    w.line("}");
    w.blankLine();

    // Body
    if (op.body) {
      w.import(`${ctx.module}/internal/utils`);
      w.import("bytes");
      w.line(
        `bodyBytes, err := utils.MarshalJSON(request.${op.body.name}, "", false)`,
      );
      w.line("if err != nil {");
      w.indent(() => w.line(`return nil, fmt.Errorf("error marshaling request: %w", err)`));
      w.line("}");
      w.line("bodyReader := bytes.NewReader(bodyBytes)");
      w.line(`req, err := http.NewRequestWithContext(ctx, ${JSON.stringify(op.verb)}, opURL, bodyReader)`);
    } else {
      w.line(`req, err := http.NewRequestWithContext(ctx, ${JSON.stringify(op.verb)}, opURL, nil)`);
    }
    w.line("if err != nil {");
    w.indent(() => w.line(`return nil, fmt.Errorf("error creating request: %w", err)`));
    w.line("}");
    if (op.body) {
      w.line(`req.Header.Set("Content-Type", ${JSON.stringify(op.body.contentType)})`);
    }
    w.line(`req.Header.Set("Accept", "application/json")`);
    w.line("req.Header.Set(\"User-Agent\", s.sdkConfiguration.UserAgent)");
    w.blankLine();

    // Query params
    if (queryParams.length > 0) {
      w.import(`${ctx.module}/internal/utils`);
      w.line("if err := utils.PopulateQueryParams(ctx, req, request, nil, nil); err != nil {");
      w.indent(() => w.line(`return nil, fmt.Errorf("error populating query params: %w", err)`));
      w.line("}");
      w.blankLine();
    }

    // Send
    w.line("httpRes, err := s.sdkConfiguration.Client.Do(req)");
    w.line("if err != nil {");
    w.indent(() => w.line(`return nil, fmt.Errorf("error sending request: %w", err)`));
    w.line("}");
    w.line("defer httpRes.Body.Close()");
    w.blankLine();

    // Build response shell
    w.import(`${ctx.module}/models/components`);
    w.line(`res := &operations.${responseStructName}{`);
    w.indent(() => {
      w.line("HTTPMeta: components.HTTPMetadata{");
      w.indent(() => {
        w.line("Request:  req,");
        w.line("Response: httpRes,");
      });
      w.line("},");
    });
    w.line("}");
    w.blankLine();

    // Status switching
    emitResponseSwitch(ctx, op, "res");

    w.line("return res, nil");
  });
  w.line("}");
}

function emitBaseURL(w: Writer): void {
  w.line(
    "baseURL = utils.ReplaceParameters(s.sdkConfiguration.GetServerDetails())",
  );
}

function emitResponseSwitch(
  ctx: FormatCtx,
  op: Operation,
  resVar: string,
): void {
  const w = ctx.writer;
  if (op.responses.length === 0) {
    w.line("// No declared responses; ignoring body.");
    return;
  }
  // Determine which imports we'll need, based on whether any case actually
  // decodes a body. The default case always reads the body via io.ReadAll.
  const anyJsonDecode = op.responses.some((r) => r.bodyType || (r.isError && r.errorTypeName));
  if (anyJsonDecode) w.import("encoding/json");
  w.import("io");
  w.line("switch httpRes.StatusCode {");
  for (const r of op.responses) {
    w.line(`case ${r.status}:`);
    w.indent(() => emitResponseCase(ctx, r, resVar));
  }
  w.line("default:");
  w.indent(() => {
    w.import(`${ctx.module}/models/apierrors`);
    w.line("body, _ := io.ReadAll(httpRes.Body)");
    w.line(
      `return nil, apierrors.NewAPIError(fmt.Sprintf("unexpected status code %d", httpRes.StatusCode), httpRes.StatusCode, string(body), httpRes)`,
    );
  });
  w.line("}");
  w.blankLine();
}

function emitResponseCase(ctx: FormatCtx, r: OperationResponse, resVar: string): void {
  const w = ctx.writer;
  if (r.isError && r.errorTypeName) {
    w.import(`${ctx.module}/models/apierrors`);
    w.line("body, err := io.ReadAll(httpRes.Body)");
    w.line("if err != nil { return nil, err }");
    w.line(`var apiErr apierrors.${r.errorTypeName}`);
    w.line("_ = json.Unmarshal(body, &apiErr)");
    w.line("apiErr.HTTPMeta = " + resVar + ".HTTPMeta");
    w.line("return nil, &apiErr");
    return;
  }
  if (!r.bodyType) {
    // Success but empty body
    w.line("// no body");
    return;
  }
  // Success body — decode and attach to response struct under a generic name.
  // The operations.<Op>Response struct will have a field named after the body type;
  // gen_operations chooses the field name. For v1 simplicity, name the response
  // body field "Body" of the appropriate type pointer.
  w.line("body, err := io.ReadAll(httpRes.Body)");
  w.line("if err != nil { return nil, err }");
  w.line(`var out ${formatType(ctx, r.bodyType)}`);
  w.line("if err := json.Unmarshal(body, &out); err != nil { return nil, err }");
  w.line(`${resVar}.Body = &out`);
}

function camel(s: string): string {
  if (!s) return s;
  return s.charAt(0).toLowerCase() + s.slice(1);
}
