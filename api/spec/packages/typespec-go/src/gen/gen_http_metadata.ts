/**
 * Emit `models/components/httpmetadata.go` — the HTTPMetadata struct attached
 * to every operation response.
 */
import { GENERATED_BANNER } from "./writer.js";

export function emitHttpMetadata(_packageName: string): { path: string; content: string } {
  const content = `${GENERATED_BANNER}

package components

import "net/http"

// HTTPMetadata carries the raw request and response for the operation.
type HTTPMetadata struct {
\tRequest  *http.Request  \`json:"-"\`
\tResponse *http.Response \`json:"-"\`
}
`;
  return { path: "models/components/httpmetadata.go", content };
}
