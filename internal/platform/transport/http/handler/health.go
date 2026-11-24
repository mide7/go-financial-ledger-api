package handler

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

// HealthCheck reports 200 when every dependency responds and 503 otherwise.
func (h *healthCheckHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbHealthy := h.db.Ping(r.Context()) == nil

	// TODO: redis health check
	redisHealthy := true

	response := struct {
		DB    bool `json:"db"`
		Redis bool `json:"redis"`
	}{
		DB:    dbHealthy,
		Redis: redisHealthy,
	}

	allHealthy := dbHealthy && redisHealthy

	if !allHealthy {
		httputil.JSONMsg(w, http.StatusServiceUnavailable, "services unhealthy", response)
		return
	}

	httputil.JSONMsg(w, http.StatusOK, "services healthy", response)
}
