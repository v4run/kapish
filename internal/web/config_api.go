package web

import (
	"encoding/json"
	"net/http"

	kconfig "github.com/v4run/kapish/internal/config"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.opts.AppConfig)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var incoming kconfig.Config
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if err := kconfig.Validate(incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.opts.AppConfig = incoming
	if s.opts.ConfigPath != "" {
		if err := kconfig.WriteToFile(s.opts.ConfigPath, incoming); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
