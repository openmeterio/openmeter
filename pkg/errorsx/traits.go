package errorsx

// Used for breaking .Handle() with a warning level log
var Warning = NewTrait("warning")

var NotFound = NewTrait("not_found")

var BadRequest = NewTrait("bad_request")

var Conflict = NewTrait("Conflict")

var Forbidden = NewTrait("forbidden")
