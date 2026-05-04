/**
 * TypeSpec emitter entry point for `@openmeter/typespec-go`.
 *
 * The emitter:
 *   1. Builds an intermediate SdkModel from the TypeSpec Program (`model/builder.ts`).
 *   2. Generates Go source via per-decl emitters in `gen/`.
 *   3. Copies the vendored static template tree (`internal/`, `retry/`, `types/`,
 *      `optionalnullable/`, `models/apierrors/apierror.go`) with the consumer's
 *      Go module path substituted for the original.
 *   4. Writes everything under `context.emitterOutputDir`.
 *
 * After writing, the emitter attempts to run `gofmt -s -w .` to normalize
 * formatting. This is best-effort; if Go isn't on PATH, a warning is reported.
 */
import { execSync } from "node:child_process";
import { mkdir, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { NoTarget, type EmitContext } from "@typespec/compiler";

import { buildSdkModel } from "./model/builder.js";
import { emitApiErrorFiles } from "./gen/gen_apierrors.js";
import { emitDeclFile } from "./gen/gen_decl.js";
import { emitGoMod } from "./gen/gen_go_mod.js";
import { emitHttpMetadata } from "./gen/gen_http_metadata.js";
import { emitOperationFiles, emitOperationsCommon } from "./gen/gen_operations.js";
import { emitReadme } from "./gen/gen_readme.js";
import { emitRootFile } from "./gen/gen_root.js";
import { emitServiceFile } from "./gen/gen_service.js";
import { emitVendoredFiles } from "./gen/gen_vendor.js";
import { GoEmitterOptions, reportDiagnostic } from "./lib.js";

interface OutFile {
  path: string;
  content: string;
}

export async function $onEmit(context: EmitContext<GoEmitterOptions>) {
  if (!context.options.module) {
    reportDiagnostic(context.program, {
      code: "missing-module",
      target: NoTarget,
    });
    return;
  }

  const sdk = buildSdkModel(context);

  // Compute the set of Go types that are string-backed (enums and aliases of
  // string). Used by formatters to pick the correct zero-value literal.
  const stringBackedTypes = new Set<string>();
  for (const decl of sdk.components) {
    if (decl.kind === "enum") stringBackedTypes.add(decl.name);
    if (decl.kind === "alias" && decl.target.kind === "scalar" && decl.target.name === "string") {
      stringBackedTypes.add(decl.name);
    }
    // Per-variant singleton enum types (Type<Variant>) are also string-backed.
    if (decl.kind === "struct" && decl.discriminatorVariant) {
      stringBackedTypes.add(decl.discriminatorVariant.singletonEnumName);
    }
    // Discriminated-union parent enum (synthesized <Name>Type) is string-backed.
    if (decl.kind === "discriminated-union" && !decl.discriminatorEnumName) {
      stringBackedTypes.add(`${decl.name}Type`);
    }
    // Heuristic-union parent enum (<Name>Type) is string-backed.
    if (decl.kind === "heuristic-union") {
      stringBackedTypes.add(`${decl.name}Type`);
    }
  }

  const files: OutFile[] = [];

  // 0. README.md (top-level docs)
  files.push(emitReadme(sdk));

  // 1. go.mod
  files.push(emitGoMod(sdk.module));

  // 2. Root SDK file
  files.push(emitRootFile(sdk));

  // 3. HTTPMetadata in components/
  files.push(emitHttpMetadata(sdk.packageName));

  // 4. Service files
  for (const svc of sdk.services) {
    files.push(emitServiceFile(sdk, svc, stringBackedTypes));
  }

  // 5. Component decls (models, enums, unions, aliases)
  for (const decl of sdk.components) {
    files.push(emitDeclFile(sdk.module, decl, stringBackedTypes));
  }

  // 6. Operation request/response structs + common Option/Options.
  files.push(emitOperationsCommon(sdk));
  for (const svc of sdk.services) {
    for (const op of svc.operations) {
      files.push(...emitOperationFiles(sdk, op, stringBackedTypes));
    }
  }

  // 7. Status-coded error types.
  files.push(...emitApiErrorFiles(sdk.module, sdk.errorStatusCodes));

  // 8. Vendored static files.
  files.push(...emitVendoredFiles(sdk.module));

  // Write everything.
  for (const f of files) {
    const full = join(context.emitterOutputDir, f.path);
    await mkdir(dirname(full), { recursive: true });
    await writeFile(full, f.content, "utf8");
  }

  // Best-effort gofmt.
  try {
    execSync("gofmt -s -w .", {
      cwd: context.emitterOutputDir,
      stdio: ["ignore", "ignore", "pipe"],
    });
  } catch {
    // gofmt unavailable; the generated files are valid Go but unformatted.
    // Not a fatal condition.
  }
}
