package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maraichr/lattice/pkg/apierr"
)

type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if h.pool != nil {
		if err := h.pool.Ping(r.Context()); err != nil {
			writeAPIError(w, nil, apierr.DatabaseNotReady())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
