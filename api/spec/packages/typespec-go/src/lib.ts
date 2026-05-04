import { createTypeSpecLibrary, JSONSchemaType, paramMessage } from "@typespec/compiler";

export interface GoEmitterOptions {
  /** Go module path written into go.mod. Required. */
  module: string;
  /** Top-level Go package name. Defaults to the last segment of `module`. */
  "package-name"?: string;
  /** SDK version string embedded as SDKVersion. */
  "sdk-version"?: string;
  /** User-Agent header default. Defaults to `openmeter-go/<sdk-version>`. */
  "user-agent"?: string;
}

const EmitterOptionsSchema: JSONSchemaType<GoEmitterOptions> = {
  type: "object",
  additionalProperties: true,
  properties: {
    module: {
      type: "string",
      description: "Go module path written into go.mod.",
    },
    "package-name": {
      type: "string",
      nullable: true,
      description: "Top-level Go package name. Defaults to last segment of module.",
    },
    "sdk-version": {
      type: "string",
      nullable: true,
      description: "SDK version embedded as SDKVersion.",
    },
    "user-agent": {
      type: "string",
      nullable: true,
      description: "Default User-Agent header.",
    },
  },
  required: ["module"],
};

export const $lib = createTypeSpecLibrary({
  name: "typespec-go",
  diagnostics: {
    "missing-module": {
      severity: "error",
      messages: {
        default:
          "The `module` emitter option is required (the Go module path written into go.mod).",
      },
    },
    "unsupported-type": {
      severity: "error",
      messages: {
        default: paramMessage`Unsupported type for Go emission: ${"kind"} (${"name"}).`,
      },
    },
    "vendor-template-missing": {
      severity: "error",
      messages: {
        default: paramMessage`Vendored template not found at ${"path"}. This is a packaging bug.`,
      },
    },
  },
  emitter: {
    options: EmitterOptionsSchema,
  },
});

export const { reportDiagnostic } = $lib;
