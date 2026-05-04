/**
 * Emit `README.md` — top-level documentation for the generated Go SDK.
 *
 * Inspired by the Speakeasy reference SDK's README but trimmed to essentials:
 * title, summary, installation, usage example, services list, server
 * selection, custom HTTP client, error handling. The exhaustive
 * "Available Resources and Operations" table (one bullet per method) is
 * skipped because for OpenMeter that would explode to ~150 lines and the
 * service/method names are discoverable via `godoc`.
 *
 * The example operation is chosen heuristically: prefer the first `GET` op
 * with no path params on the first service. Falls back to any op if none fits.
 */
import type { Operation, SdkModel, Service } from "../model/index.js";

export function emitReadme(sdk: SdkModel): { path: string; content: string } {
  const lines: string[] = [];
  lines.push(`# ${sdk.packageName}`);
  lines.push("");
  lines.push(`Type-safe Go SDK for the ${sdk.title}.`);
  lines.push("");

  if (sdk.summary) {
    lines.push("## Summary");
    lines.push("");
    lines.push(stripBlankSuffix(sdk.summary));
    lines.push("");
  }

  // Table of contents (terse)
  lines.push("## Table of Contents");
  lines.push("");
  lines.push("* [Installation](#installation)");
  lines.push("* [Example](#example)");
  lines.push("* [Services](#services)");
  lines.push("* [Server selection](#server-selection)");
  lines.push("* [Custom HTTP client](#custom-http-client)");
  lines.push("* [Error handling](#error-handling)");
  lines.push("");

  // Installation
  lines.push("## Installation");
  lines.push("");
  lines.push("```bash");
  lines.push(`go get ${sdk.module}`);
  lines.push("```");
  lines.push("");

  // Example
  lines.push("## Example");
  lines.push("");
  lines.push("```go");
  lines.push(...renderExample(sdk).split("\n"));
  lines.push("```");
  lines.push("");

  // Services
  lines.push("## Services");
  lines.push("");
  if (sdk.services.length === 0) {
    lines.push("_No services declared in this SDK._");
  } else {
    for (const s of sdk.services) {
      const summary = s.operations.length === 1
        ? `1 operation`
        : `${s.operations.length} operations`;
      lines.push(`* \`${s.rootFieldName}\` — ${summary}`);
    }
  }
  lines.push("");

  // Server selection
  lines.push("## Server selection");
  lines.push("");
  if (sdk.servers.length === 0) {
    lines.push("No servers are declared in the spec; use `WithServerURL` to point the SDK at your deployment:");
  } else {
    lines.push("The SDK ships with the following server URLs (selected via `WithServerIndex`):");
    lines.push("");
    sdk.servers.forEach((s, i) => {
      const desc = s.description ? ` — ${s.description}` : "";
      lines.push(`${i}. \`${s.url}\`${desc}`);
    });
    lines.push("");
    lines.push("You can also pass a custom URL with `WithServerURL`:");
  }
  lines.push("");
  lines.push("```go");
  lines.push(`s := ${sdk.packageName}.New(${sdk.packageName}.WithServerURL("https://api.example.com"))`);
  lines.push("```");
  lines.push("");

  // Custom HTTP client
  lines.push("## Custom HTTP client");
  lines.push("");
  lines.push(
    "Provide any value implementing the `HTTPClient` interface (`Do(*http.Request) (*http.Response, error)`):",
  );
  lines.push("");
  lines.push("```go");
  lines.push("httpClient := &http.Client{Timeout: 30 * time.Second}");
  lines.push(`s := ${sdk.packageName}.New(${sdk.packageName}.WithClient(httpClient))`);
  lines.push("```");
  lines.push("");

  // Error handling
  lines.push("## Error handling");
  lines.push("");
  lines.push("All operations return `(*operations.XxxResponse, error)`. Status-coded errors are typed:");
  lines.push("");
  lines.push("```go");
  lines.push("res, err := s.SomeService.SomeOperation(ctx, req)");
  lines.push("if err != nil {");
  lines.push("    var notFound *apierrors.NotFoundError");
  lines.push("    if errors.As(err, &notFound) {");
  lines.push("        // resource missing");
  lines.push("    }");
  lines.push("    return err");
  lines.push("}");
  lines.push("```");
  lines.push("");

  lines.push("---");
  lines.push("");
  lines.push("_This SDK was generated from a TypeSpec definition. Do not edit generated files directly._");
  lines.push("");

  return {
    path: "README.md",
    content: lines.join("\n"),
  };
}

function stripBlankSuffix(s: string): string {
  return s.replace(/\s+$/, "");
}

function renderExample(sdk: SdkModel): string {
  const example = pickExampleOperation(sdk);
  if (!example) {
    return `package main

import (
\t"context"
\t"log"

\t"${sdk.module}"
)

func main() {
\tctx := context.Background()
\ts := ${sdk.packageName}.New()
\t_ = ctx
\t_ = s
\tlog.Println("SDK initialized")
}`;
  }

  const { service, op } = example;
  const noArgs = op.params.every((p) => p.kind !== "path") && op.body === undefined;
  const callArgs = noArgs
    ? "ctx"
    : `ctx /* TODO: fill in args for ${op.methodName} */`;
  return `package main

import (
\t"context"
\t"log"

\t${sdk.packageName} "${sdk.module}"
)

func main() {
\tctx := context.Background()
\ts := ${sdk.packageName}.New()

\tres, err := s.${service.rootFieldName}.${op.methodName}(${callArgs})
\tif err != nil {
\t\tlog.Fatal(err)
\t}
\t_ = res
}`;
}

function pickExampleOperation(
  sdk: SdkModel,
): { service: Service; op: Operation } | undefined {
  // Prefer a parameterless GET on the first service that has one.
  for (const s of sdk.services) {
    const op = s.operations.find(
      (o) =>
        o.verb === "GET" &&
        o.params.every((p) => p.kind !== "path") &&
        o.body === undefined,
    );
    if (op) return { service: s, op };
  }
  // Fall back to any GET.
  for (const s of sdk.services) {
    const op = s.operations.find((o) => o.verb === "GET");
    if (op) return { service: s, op };
  }
  // Fall back to anything.
  for (const s of sdk.services) {
    const op = s.operations[0];
    if (op) return { service: s, op };
  }
  return undefined;
}
