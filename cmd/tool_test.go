package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{name: "empty", profile: "", wantErr: true},
		{name: "spaces", profile: "   ", wantErr: true},
		{name: "dotfile", profile: ".work", wantErr: true},
		{name: "path", profile: "a/b", wantErr: true},
		{name: "ok", profile: "work", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProfileName(tc.profile)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestClaudeLifecycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	status, err := currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom>, got %q", status)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	status, err = currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus after switch: %v", err)
	}
	if status != "work" {
		t.Fatalf("expected work, got %q", status)
	}

	if err := os.WriteFile(configPath, []byte(`{"x":2}`), 0o600); err != nil {
		t.Fatalf("write config (modified): %v", err)
	}

	status, err = currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus after modify: %v", err)
	}
	if status != "work (modified)" {
		t.Fatalf("expected work (modified), got %q", status)
	}

	cleared, err := deleteProfile(cfg, "work")
	if err != nil {
		t.Fatalf("deleteProfile: %v", err)
	}
	if !cleared {
		t.Fatalf("expected deleting active profile to clear current")
	}

	status, err = currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus after delete: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom>, got %q", status)
	}
}

func TestSaveProfileWithoutForceFailsIfExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("first saveProfile: %v", err)
	}
	if err := saveProfile(cfg, "work", false); err == nil {
		t.Fatalf("expected error on second save without --force, got nil")
	}
	if err := saveProfile(cfg, "work", true); err != nil {
		t.Fatalf("saveProfile with --force: %v", err)
	}
}

func TestExecuteDoesNotDuplicateErrors(t *testing.T) {
	oldOut := rootCmd.OutOrStdout()
	oldErr := rootCmd.ErrOrStderr()
	t.Cleanup(func() {
		rootCmd.SetOut(oldOut)
		rootCmd.SetErr(oldErr)
		rootCmd.SetArgs(nil)
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"does-not-exist"})

	err := Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	got := stderr.String()
	if strings.Count(got, `unknown command "does-not-exist"`) != 1 {
		t.Fatalf("expected error printed once, got stderr:\n%s", got)
	}
}

func TestCurrentStatusRejectsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}
	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	target := filepath.Join(home, "real-settings.json")
	if err := os.WriteFile(target, []byte(`{"x":2}`), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("remove config: %v", err)
	}
	if err := os.Symlink(target, configPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if _, err := currentStatus(cfg); err == nil || !errors.Is(err, errSymlinkNotAllowed) {
		t.Fatalf("expected symlink error, got %v", err)
	}
}
