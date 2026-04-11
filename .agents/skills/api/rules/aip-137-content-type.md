# AIP-137 — Content-Type

Reference: https://kong-aip.netlify.app/aip/137/

## Default content type

When the `Content-Type` header is absent, it defaults to `application/json; charset=utf-8` (note: the default **includes** a charset). APIs must not reject valid JSON payloads that omit the header.

## Validation

Validate the request body's `Content-Type` on `POST`, `PUT`, and `PATCH`. Validation fails when:

- The `Content-Type` header specifies an unsupported type, **or**
- The body does not match the declared type

Either failure mode returns `415 Unsupported Media Type`. When responding with 415, include an `Accept-{METHOD}` header listing supported types (e.g., `Accept-POST: application/json`). If the rejection was caused by a charset mismatch, the charset directive must appear in the `Accept-{METHOD}` value.

## Exclusions

**Endpoints without request bodies are exempt** from content-type validation — there is nothing to validate.
