package main

import (
	"fmt"
	"net/http"
	"os"
)

// path: /admin/metrics
func (cfg *apiConfig) showMetricsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	hitCount := cfg.fileserverHits.Load()
	response := fmt.Sprintf(`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`, hitCount)
	w.Header().Values("Content-Type: text/html; charset=utf-8;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// path: /admin/reset
func (cfg *apiConfig) resetMetricHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if os.Getenv("PLATFROM") == "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(http.StatusText(http.StatusForbidden)))
		return
	}

	cfg.resetMetrics()
	// delete all users from the database
	cfg.database.DeleteAllUsers(r.Context())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metrics have been reset!"))
}
