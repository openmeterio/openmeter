package openmeter

import "time"

// Pointer helpers for populating optional request fields inline. They mirror the
// convention used by other Go cloud SDKs (e.g. aws.String), keeping call sites
// free of one-off address-of locals.

// String returns a pointer to s.
func String(s string) *string { return &s }

// Int returns a pointer to i.
func Int(i int) *int { return &i }

// Bool returns a pointer to b.
func Bool(b bool) *bool { return &b }

// Time returns a pointer to t.
func Time(t time.Time) *time.Time { return &t }
