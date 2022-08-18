package config

import (
	"encoding/json"
	"fmt"
	"net/http"

	"albumart.digital/jsonlog"
)

type Handler struct {
}

func (*Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(struct {
			Message string `json:"message"`
		}{"Only the GET method is supported on this endpoint"})

		jsonlog.Level("INFO").
			Field("address", r.RemoteAddr).
			Field("status", http.StatusMethodNotAllowed).
			Field("method", r.Method).
			Msg("invalid method")
		return
	}

	if value, ok := r.URL.Query()["loglevel"]; ok {
		if len(value) != 1 {
			return
		}

		jsonlog.SetLevel(r.URL.Query().Get("loglevel"))
		jsonlog.Level("INFO").Msg(
			fmt.Sprintf(
				"log level set to %v",
				r.URL.Query().Get("loglevel"),
			))
	}
}
