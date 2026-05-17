import * as ay from "@alloy-js/core";
import * as ts from "@alloy-js/typescript";
import { type EmitContext, ListenerFlow, navigateProgram, type Program } from "@typespec/compiler";
import { $ } from "@typespec/compiler/typekit";
import { Output, writeOutput } from "@typespec/emitter-framework";
import { ZodSchemaDeclaration } from "./components/ZodSchemaDeclaration.jsx";
import { zod } from "./external-packages/zod.js";
import { newTopologicalTypeCollector } from "./utils.jsx";

export async function $onEmit(context: EmitContext) {
  const types = getAllDataTypes(context.program);
  const tsNamePolicy = ts.createTSNamePolicy();

  writeOutput(
    context.program,
    <Output program={context.program} namePolicy={tsNamePolicy} externals={[zod]}>
      <ts.SourceFile path="models.ts">
        <ay.For
          each={types}
          ender={";"}
          joiner={
            <>
              ;<hbr />
              <hbr />
            </>
          }
        >
          {(type) => <ZodSchemaDeclaration type={type} export />}
        </ay.For>
      </ts.SourceFile>
    </Output>,
    context.emitterOutputDir,
  );
}

/**
 * Collects all the models defined in the spec and returns them in
 * topologically sorted order. Types are ordered such that dependencies appear
 * before the types that depend on them.
 */
function getAllDataTypes(program: Program) {
  const collector = newTopologicalTypeCollector(program);
  const globalNs = program.getGlobalNamespaceType();

  navigateProgram(
    program,
    {
      namespace(n) {
        if (n !== globalNs && !$(program).type.isUserDefined(n)) {
          return ListenerFlow.NoRecursion;
        }
        return undefined;
      },
      model: collector.collectType,
      enum: collector.collectType,
      union: collector.collectType,
      scalar: collector.collectType,
    },
    { includeTemplateDeclaration: false },
  );

  return collector.types;
}
