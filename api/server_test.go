package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"tokyo/pkg/profile"
)

func TestListProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := profile.ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := profile.Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	server := NewServer()
	req := httptest.NewRequest("GET", "/api/claude/profiles", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp["profiles"]) != 1 || resp["profiles"][0] != "work" {
		t.Fatalf("expected [work], got %v", resp["profiles"])
	}
}

func TestCurrentStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	server := NewServer()
	req := httptest.NewRequest("GET", "/api/claude/current", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["custom"] != true {
		t.Fatalf("expected custom=true, got %v", resp)
	}
}

func TestSaveProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	server := NewServer()
	body := bytes.NewBufferString(`{"profile":"work"}`)
	req := httptest.NewRequest("POST", "/api/claude/profiles", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	exists, _ := profile.Exists(profile.ClaudeTool(), "work")
	if !exists {
		t.Fatalf("profile should exist")
	}
}

func TestSaveProfileConflict(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := profile.ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := profile.Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	server := NewServer()
	body := bytes.NewBufferString(`{"profile":"work"}`)
	req := httptest.NewRequest("POST", "/api/claude/profiles", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSwitchProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := profile.ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := profile.Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	server := NewServer()
	req := httptest.NewRequest("POST", "/api/claude/switch/work", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status, _ := profile.Current(tool)
	if status != "work" {
		t.Fatalf("expected work, got %s", status)
	}
}

func TestSwitchProfileNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := NewServer()
	req := httptest.NewRequest("POST", "/api/claude/switch/nonexistent", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := profile.ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := profile.Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	server := NewServer()
	req := httptest.NewRequest("DELETE", "/api/claude/profiles/work", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	exists, _ := profile.Exists(tool, "work")
	if exists {
		t.Fatalf("profile should not exist")
	}
}

func TestUnknownTool(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest("GET", "/api/unknown/profiles", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInvalidProfileName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := NewServer()
	body := bytes.NewBufferString(`{"profile":""}`)
	req := httptest.NewRequest("POST", "/api/claude/profiles", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
