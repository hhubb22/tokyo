package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tokyo/pkg/profile"
)

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

func TestListCommandOutput(t *testing.T) {
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

	cmd := newListCommand(tool)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("list command: %v", err)
	}

	if !strings.Contains(out.String(), "work") {
		t.Fatalf("expected 'work' in output, got %q", out.String())
	}
}

func TestCurrentCommandOutput(t *testing.T) {
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

	cmd := newCurrentCommand(tool)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("current command: %v", err)
	}

	if !strings.Contains(out.String(), "<custom>") {
		t.Fatalf("expected '<custom>' in output, got %q", out.String())
	}
}

func TestDeleteCommandOutput(t *testing.T) {
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
	if err := profile.Switch(tool, "work"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	cmd := newDeleteCommand(tool)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"work"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete command: %v", err)
	}

	if !strings.Contains(out.String(), "<custom>") {
		t.Fatalf("expected '<custom>' message in output, got %q", out.String())
	}
}

func TestSwitchCommandSuccess(t *testing.T) {
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

	cmd := newSwitchCommand(tool)
	cmd.SetArgs([]string{"work"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("switch command: %v", err)
	}

	status, _ := profile.Current(tool)
	if status != "work" {
		t.Fatalf("expected work, got %q", status)
	}
}
