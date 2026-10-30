package main

import "net/http"

// storageErrorStatus maps provider failures to HTTP status codes. Unimplemented
// network/cloud providers must answer 501 rather than pretending to be a
// transient server error, so clients can show an explicit unavailable state.
func (s *Server) storageError(w http.ResponseWriter, code string, err error) {
	if isNotImplemented(err) {
		s.error(w, http.StatusNotImplemented, "not_implemented", "storage provider is not implemented")
		return
	}
	s.error(w, http.StatusBadRequest, code, err.Error())
}
