package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"tokyo/pkg/profile"
)

type Server struct {
	mux   *http.ServeMux
	tools map[string]profile.Tool
}

func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
		tools: map[string]profile.Tool{
			"claude": profile.ClaudeTool(),
			"codex":  profile.CodexTool(),
		},
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/{tool}/profiles", s.handleList)
	s.mux.HandleFunc("GET /api/{tool}/current", s.handleCurrent)
	s.mux.HandleFunc("POST /api/{tool}/profiles", s.handleSave)
	s.mux.HandleFunc("POST /api/{tool}/switch/{profile}", s.handleSwitch)
	s.mux.HandleFunc("DELETE /api/{tool}/profiles/{profile}", s.handleDelete)
}

func (s *Server) getTool(r *http.Request) (profile.Tool, bool) {
	toolName := r.PathValue("tool")
	tool, ok := s.tools[toolName]
	return tool, ok
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	tool, ok := s.getTool(r)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	profiles, err := profile.List(tool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

func (s *Server) handleCurrent(w http.ResponseWriter, r *http.Request) {
	tool, ok := s.getTool(r)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	status, err := profile.Current(tool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	modified := strings.HasSuffix(status, " (modified)")
	name := strings.TrimSuffix(status, " (modified)")
	custom := name == "<custom>"

	writeJSON(w, http.StatusOK, map[string]any{
		"profile":  name,
		"modified": modified,
		"custom":   custom,
	})
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	tool, ok := s.getTool(r)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	var req struct {
		Profile string `json:"profile"`
		Force   bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := profile.ValidateProfileName(req.Profile); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := profile.Save(tool, req.Profile, req.Force); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"profile": req.Profile})
}

func (s *Server) handleSwitch(w http.ResponseWriter, r *http.Request) {
	tool, ok := s.getTool(r)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	profileName := r.PathValue("profile")
	if err := profile.ValidateProfileName(profileName); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := profile.Switch(tool, profileName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"profile": profileName})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	tool, ok := s.getTool(r)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	profileName := r.PathValue("profile")
	if err := profile.ValidateProfileName(profileName); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleared, err := profile.Delete(tool, profileName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"cleared": cleared})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
