/**
 * Emit per-status-code error types under `models/apierrors/`.
 *
 * Each error type is a struct embedding HTTPMeta. The error body shape depends
 * on the spec; for v1, we use the standard RFC 7807 fields (problem+json),
 * matching what the OpenMeter spec returns.
 */
import { GENERATED_BANNER, Writer } from "./writer.js";

export interface ApiErrorFile {
  readonly path: string;
  readonly content: string;
}

const APIERRORS_PKG = "models/apierrors";

const STATUS_TYPE_NAMES: Record<number, string> = {
  400: "BadRequestError",
  401: "UnauthorizedError",
  403: "ForbiddenError",
  404: "NotFoundError",
  409: "ConflictError",
  410: "GoneError",
  413: "PayloadTooLargeError",
  415: "UnsupportedMediaTypeError",
  422: "UnprocessableContentError",
  429: "TooManyRequestsError",
  500: "InternalError",
  501: "NotImplementedError",
  503: "ServiceUnavailableError",
};

export function statusTypeName(status: number): string {
  return STATUS_TYPE_NAMES[status] ?? `APIError${status}`;
}

export function emitApiErrorFiles(module: string, statuses: readonly number[]): ApiErrorFile[] {
  const out: ApiErrorFile[] = [];
  for (const status of statuses) {
    const name = statusTypeName(status);
    out.push(emitApiErrorFile(module, name));
  }
  return out;
}

function emitApiErrorFile(module: string, name: string): ApiErrorFile {
  const w = new Writer();
  w.preamble(GENERATED_BANNER);
  w.packageName("apierrors");
  w.import(`${module}/models/components`);
  w.import("encoding/json");

  w.line(`type ${name} struct {`);
  w.indent(() => {
    w.line("Type     *string                 `json:\"type,omitzero\"`");
    w.line("Title    *string                 `json:\"title,omitzero\"`");
    w.line("Status   *int                    `json:\"status,omitzero\"`");
    w.line("Detail   *string                 `json:\"detail,omitzero\"`");
    w.line("Instance *string                 `json:\"instance,omitzero\"`");
    w.line("HTTPMeta components.HTTPMetadata `json:\"-\"`");
  });
  w.line("}");
  w.blankLine();

  w.line(`var _ error = &${name}{}`);
  w.blankLine();

  w.line(`func (e *${name}) Error() string {`);
  w.indent(() => {
    w.line("data, _ := json.Marshal(e)");
    w.line("return string(data)");
  });
  w.line("}");

  return {
    path: `${APIERRORS_PKG}/${name.toLowerCase()}.go`,
    content: w.finish(),
  };
}
