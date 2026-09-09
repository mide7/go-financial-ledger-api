package handlers

import (
	"net/http"
)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {

	// TODO: db health check
	// TODO: redis health check

	// TODO: add more health checks

}
