// Package definitions lists the Notification Event type definitions
// of every producing domain. Definitions are generated from the domains'
// TypeSpec contracts; run `make gen-api` after adding or changing one. It is a
// separate package because the domain packages import v4, so v4 cannot import
// them.
//
// definitions.sum records the fingerprint and replacement of every definition,
// so that changing a released definition fails Verify instead of slipping
// through review.
package definitions
