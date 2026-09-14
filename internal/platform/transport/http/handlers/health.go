package handlers

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

func (h *handler) HealthCheck(w http.ResponseWriter, r *http.Request) {

	var dbHealthy bool = true
	if err := h.dbPool.Ping(r.Context()); err != nil {
		dbHealthy = false
	}
	// TODO: redis health check
	var redisHealthy bool = true

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
