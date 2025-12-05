package oasmiddleware

import (
	"bytes"
	"net/http"
)

// ResponseWriterWrapper leverage to the original response write to be able to access
// the written body and the status code of the response. This is used in the response middleware
// but can also be used in a logging middleware to log the status code of the response.
type ResponseWriterWrapper struct {
	w          *http.ResponseWriter
	body       *bytes.Buffer
	statusCode *int
}

func (rww ResponseWriterWrapper) Body() *bytes.Buffer {
	return rww.body
}

func (rww ResponseWriterWrapper) StatusCode() *int {
	return rww.statusCode
}

func NewResponseWriterWrapper(w http.ResponseWriter) ResponseWriterWrapper {
	var (
		buf        bytes.Buffer
		statusCode = 200
	)
	return ResponseWriterWrapper{
		w:          &w,
		body:       &buf,
		statusCode: &statusCode,
	}
}

// Write function overwrites the http.ResponseWriter Header() function
func (rww ResponseWriterWrapper) Write(buf []byte) (int, error) {
	rww.body.Write(buf)
	return (*rww.w).Write(buf)
}

// Header function overwrites the http.ResponseWriter Header() function
func (rww ResponseWriterWrapper) Header() http.Header {
	return (*rww.w).Header()
}

// WriteHeader function overwrites the http.ResponseWriter WriteHeader() function
func (rww ResponseWriterWrapper) WriteHeader(statusCode int) {
	*rww.statusCode = statusCode
	(*rww.w).WriteHeader(statusCode)
}

func (rww ResponseWriterWrapper) Unwrap() http.ResponseWriter {
	return *rww.w
}
