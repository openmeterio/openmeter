package router

import (
	"net/http"
)

// POST /api/v1/subjects
func (a *Router) UpsertSubject(w http.ResponseWriter, r *http.Request) {
	a.subjectHandler.UpsertSubject().ServeHTTP(w, r)
}

// GET /api/v1/subjects
func (a *Router) ListSubjects(w http.ResponseWriter, r *http.Request) {
	a.subjectHandler.ListSubjects().ServeHTTP(w, r)
}

// GET /api/v1/subjects/{subjectIdOrKey}
func (a *Router) GetSubject(w http.ResponseWriter, r *http.Request, subjectIdOrKey string) {
	a.subjectHandler.GetSubject().With(subjectIdOrKey).ServeHTTP(w, r)
}

// DELETE /api/v1/subjects/{subjectIdOrKey}
func (a *Router) DeleteSubject(w http.ResponseWriter, r *http.Request, subjectIdOrKey string) {
	a.subjectHandler.DeleteSubject().With(subjectIdOrKey).ServeHTTP(w, r)
}
