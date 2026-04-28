// Project-root marker for httpYac. The presence of this file scopes
// httpYac's project root to e2e/http/.
//
// Configuration is intentionally shell-env based, mirroring the Go
// e2e convention (`OPENMETER_ADDRESS=http://localhost:8888 go test
// ./e2e/...`). Each .http file declares
//   @api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3
// at the top — `OPENMETER_ADDRESS` stays the bare host (matches the
// /e2e Go suite exactly) and the version prefix `/api/v3` lives in
// the file. httpYac evaluates `{{...}}` as a JS expression when no
// variable matches by name, so `process.env.X` resolves to the
// shell-exported value at send time. Bare `{{OPENMETER_ADDRESS}}`
// would throw ReferenceError.
//
// Add response-log scrubbing here if real auth tokens flow through
// tests. v1 ships without scrubbing; add intentionally when the auth
// surface is settled.

module.exports = {};
