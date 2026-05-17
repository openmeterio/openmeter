import { createTypeSpecLibrary } from "@typespec/compiler";

export const $lib = createTypeSpecLibrary({
  name: "typespec-zod",
  diagnostics: {},
});

export const { reportDiagnostic, createDiagnostic } = $lib;
