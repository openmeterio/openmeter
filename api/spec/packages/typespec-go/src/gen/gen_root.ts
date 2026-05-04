/**
 * Emit `openmeter.go` — the root SDK struct, options, and `New()` constructor.
 *
 * This is a thin v1: the structure mirrors the reference SDK but only the
 * essentials are emitted. Retry and timeout options are wired through to
 * SDKConfiguration; hook plumbing is included but operations skip the calls
 * (see gen_method.ts).
 */
import type { SdkModel } from "../model/index.js";
import { GENERATED_BANNER, Writer } from "./writer.js";

export function emitRootFile(sdk: SdkModel): { path: string; content: string } {
  const w = new Writer();
  w.preamble(GENERATED_BANNER);
  w.packageName(sdk.packageName);

  w.import(`${sdk.module}/internal/config`);
  w.import(`${sdk.module}/internal/hooks`);
  w.import(`${sdk.module}/internal/utils`);
  w.import(`${sdk.module}/retry`);
  w.import("fmt");
  w.import("net/http");
  w.import("time");

  // ServerList
  w.line("// ServerList contains the list of servers available to the SDK");
  w.line("var ServerList = []string{");
  w.indent(() => {
    if (sdk.servers.length === 0) {
      // Fallback default — keeps the SDK usable when no @server is declared.
      w.line(`"https://api.example.com",`);
    } else {
      for (const s of sdk.servers) {
        if (s.description) w.line(`// ${s.description}`);
        w.line(`${JSON.stringify(s.url)},`);
      }
    }
  });
  w.line("}");
  w.blankLine();

  // HTTPClient interface
  w.line("// HTTPClient provides an interface for supplying the SDK with a custom HTTP client");
  w.line("type HTTPClient interface {");
  w.indent(() => w.line("Do(req *http.Request) (*http.Response, error)"));
  w.line("}");
  w.blankLine();

  // Helper functions (pointer creators)
  w.line("// String provides a helper function to return a pointer to a string");
  w.line("func String(s string) *string { return &s }");
  w.blankLine();
  w.line("// Bool provides a helper function to return a pointer to a bool");
  w.line("func Bool(b bool) *bool { return &b }");
  w.blankLine();
  w.line("// Int provides a helper function to return a pointer to an int");
  w.line("func Int(i int) *int { return &i }");
  w.blankLine();
  w.line("// Int64 provides a helper function to return a pointer to an int64");
  w.line("func Int64(i int64) *int64 { return &i }");
  w.blankLine();
  w.line("// Float32 provides a helper function to return a pointer to a float32");
  w.line("func Float32(f float32) *float32 { return &f }");
  w.blankLine();
  w.line("// Float64 provides a helper function to return a pointer to a float64");
  w.line("func Float64(f float64) *float64 { return &f }");
  w.blankLine();
  w.line("// Pointer provides a helper function to return a pointer to a type");
  w.line("func Pointer[T any](v T) *T { return &v }");
  w.blankLine();

  // Root SDK struct
  w.line(`// ${sdk.packageName.charAt(0).toUpperCase() + sdk.packageName.slice(1)} is the root SDK type.`);
  w.line("type OpenMeter struct {");
  w.indent(() => {
    w.line("SDKVersion string");
    for (const s of sdk.services) {
      w.line(`${s.rootFieldName} *${s.structName}`);
    }
    w.blankLine();
    w.line("sdkConfiguration config.SDKConfiguration");
    w.line("hooks            *hooks.Hooks");
  });
  w.line("}");
  w.blankLine();

  // SDKOption type
  w.line("type SDKOption func(*OpenMeter)");
  w.blankLine();

  // WithServerURL
  w.line("// WithServerURL allows providing an alternative server URL");
  w.line("func WithServerURL(serverURL string) SDKOption {");
  w.indent(() => {
    w.line("return func(sdk *OpenMeter) {");
    w.indent(() => w.line("sdk.sdkConfiguration.ServerURL = serverURL"));
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // WithTemplatedServerURL
  w.line("// WithTemplatedServerURL allows overriding the default server URL with a templated URL populated from params");
  w.line("func WithTemplatedServerURL(serverURL string, params map[string]string) SDKOption {");
  w.indent(() => {
    w.line("return func(sdk *OpenMeter) {");
    w.indent(() => {
      w.line("if params != nil {");
      w.indent(() => w.line("serverURL = utils.ReplaceParameters(serverURL, params)"));
      w.line("}");
      w.line("sdk.sdkConfiguration.ServerURL = serverURL");
    });
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // WithServerIndex
  w.line("// WithServerIndex selects a server by index from ServerList");
  w.line("func WithServerIndex(serverIndex int) SDKOption {");
  w.indent(() => {
    w.line("return func(sdk *OpenMeter) {");
    w.indent(() => {
      w.line("if serverIndex < 0 || serverIndex >= len(ServerList) {");
      w.indent(() => w.line(`panic(fmt.Errorf("server index %d out of range", serverIndex))`));
      w.line("}");
      w.line("sdk.sdkConfiguration.ServerIndex = serverIndex");
    });
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // WithClient
  w.line("// WithClient overrides the default HTTP client");
  w.line("func WithClient(client HTTPClient) SDKOption {");
  w.indent(() => {
    w.line("return func(sdk *OpenMeter) {");
    w.indent(() => w.line("sdk.sdkConfiguration.Client = client"));
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // WithRetryConfig
  w.line("// WithRetryConfig sets the retry configuration for the SDK");
  w.line("func WithRetryConfig(retryConfig retry.Config) SDKOption {");
  w.indent(() => {
    w.line("return func(sdk *OpenMeter) {");
    w.indent(() => w.line("sdk.sdkConfiguration.RetryConfig = &retryConfig"));
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // WithTimeout
  w.line("// WithTimeout sets a request timeout applied to each operation");
  w.line("func WithTimeout(timeout time.Duration) SDKOption {");
  w.indent(() => {
    w.line("return func(sdk *OpenMeter) {");
    w.indent(() => w.line("sdk.sdkConfiguration.Timeout = &timeout"));
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // New
  w.line("// New creates a new instance of the SDK with the provided options");
  w.line("func New(opts ...SDKOption) *OpenMeter {");
  w.indent(() => {
    w.line("sdk := &OpenMeter{");
    w.indent(() => {
      w.line(`SDKVersion: ${JSON.stringify(sdk.sdkVersion)},`);
      w.line("sdkConfiguration: config.SDKConfiguration{");
      w.indent(() => {
        w.line(`UserAgent:  ${JSON.stringify(sdk.userAgent)},`);
        w.line("ServerList: ServerList,");
        w.line("ServerVariables: []map[string]string{");
        w.indent(() => emitServerVariables(w, sdk));
        w.line("},");
      });
      w.line("},");
      w.line("hooks: hooks.New(),");
    });
    w.line("}");
    w.blankLine();
    w.line("for _, opt := range opts {");
    w.indent(() => w.line("opt(sdk)"));
    w.line("}");
    w.blankLine();
    w.line("if sdk.sdkConfiguration.Client == nil {");
    w.indent(() => w.line("sdk.sdkConfiguration.Client = &http.Client{Timeout: 60 * time.Second}"));
    w.line("}");
    w.blankLine();
    w.line("sdk.sdkConfiguration = sdk.hooks.SDKInit(sdk.sdkConfiguration)");
    w.blankLine();
    for (const s of sdk.services) {
      w.line(`sdk.${s.rootFieldName} = ${s.ctorName}(sdk, sdk.sdkConfiguration, sdk.hooks)`);
    }
    w.blankLine();
    w.line("return sdk");
  });
  w.line("}");

  return { path: `${sdk.packageName}.go`, content: w.finish() };
}

function emitServerVariables(w: Writer, sdk: SdkModel): void {
  if (sdk.servers.length === 0) {
    w.line("{},");
    return;
  }
  for (const s of sdk.servers) {
    if (s.variables.length === 0) {
      w.line("{},");
      continue;
    }
    w.line("{");
    w.indent(() => {
      for (const v of s.variables) {
        w.line(`${JSON.stringify(v.name)}: ${JSON.stringify(v.default ?? "")},`);
      }
    });
    w.line("},");
  }
}
