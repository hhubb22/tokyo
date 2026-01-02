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

func TestCodexLifecycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := codexConfig()
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(codexDir, "config.toml")
	authPath := filepath.Join(codexDir, "auth.json")
	if err := os.WriteFile(configPath, []byte(`key = "value1"`), 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}
	if err := os.WriteFile(authPath, []byte(`{"token":"abc"}`), 0o600); err != nil {
		t.Fatalf("write auth.json: %v", err)
	}

	// Initial status should be <custom>
	status, err := currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom>, got %q", status)
	}

	// Save profile
	if err := saveProfile(cfg, "personal", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	// Switch to profile
	if err := switchProfile(cfg, "personal"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	status, err = currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus after switch: %v", err)
	}
	if status != "personal" {
		t.Fatalf("expected personal, got %q", status)
	}

	// Modify one file - should show modified
	if err := os.WriteFile(configPath, []byte(`key = "value2"`), 0o600); err != nil {
		t.Fatalf("write config.toml (modified): %v", err)
	}

	status, err = currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus after modify: %v", err)
	}
	if status != "personal (modified)" {
		t.Fatalf("expected personal (modified), got %q", status)
	}

	// Switch back should restore both files
	if err := switchProfile(cfg, "personal"); err != nil {
		t.Fatalf("switchProfile again: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if string(data) != `key = "value1"` {
		t.Fatalf("expected original config, got %q", string(data))
	}
}

func TestListProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Empty list initially
	profiles, err := listProfiles(cfg)
	if err != nil {
		t.Fatalf("listProfiles: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected empty list, got %v", profiles)
	}

	// Save some profiles
	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile work: %v", err)
	}
	if err := saveProfile(cfg, "personal", false); err != nil {
		t.Fatalf("saveProfile personal: %v", err)
	}
	if err := saveProfile(cfg, "alpha", false); err != nil {
		t.Fatalf("saveProfile alpha: %v", err)
	}

	profiles, err = listProfiles(cfg)
	if err != nil {
		t.Fatalf("listProfiles after save: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(profiles))
	}
	// Should be sorted
	if profiles[0] != "alpha" || profiles[1] != "personal" || profiles[2] != "work" {
		t.Fatalf("expected sorted [alpha personal work], got %v", profiles)
	}
}

func TestDeleteNonExistentProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()

	_, err := deleteProfile(cfg, "nonexistent")
	if err == nil {
		t.Fatalf("expected error deleting nonexistent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestDeleteNonActiveProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile work: %v", err)
	}
	if err := saveProfile(cfg, "personal", false); err != nil {
		t.Fatalf("saveProfile personal: %v", err)
	}
	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	// Delete non-active profile
	cleared, err := deleteProfile(cfg, "personal")
	if err != nil {
		t.Fatalf("deleteProfile: %v", err)
	}
	if cleared {
		t.Fatalf("expected cleared=false for non-active profile")
	}

	// Current should still be work
	status, err := currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus: %v", err)
	}
	if status != "work" {
		t.Fatalf("expected work, got %q", status)
	}
}

func TestSwitchToNonExistentProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()

	err := switchProfile(cfg, "nonexistent")
	if err == nil {
		t.Fatalf("expected error switching to nonexistent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestSaveProfileMissingConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()

	err := saveProfile(cfg, "work", false)
	if err == nil {
		t.Fatalf("expected error saving without config file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestSwitchProfileMissingProfileFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	// Delete the profile's config file
	profileDir, _ := cfg.profileDir("work")
	profileFile := filepath.Join(profileDir, "settings.json")
	if err := os.Remove(profileFile); err != nil {
		t.Fatalf("remove profile file: %v", err)
	}

	err := switchProfile(cfg, "work")
	if err == nil {
		t.Fatalf("expected error switching with missing profile file")
	}
	if !strings.Contains(err.Error(), "missing file") {
		t.Fatalf("expected 'missing file' error, got %v", err)
	}
}

func TestProfileMatchesMissingConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	// Remove the current config file
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("remove config: %v", err)
	}

	match, err := profileMatches(cfg, "work")
	if err != nil {
		t.Fatalf("profileMatches: %v", err)
	}
	if match {
		t.Fatalf("expected no match when config file missing")
	}
}

func TestFilesEqualDifferentSizes(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")

	if err := os.WriteFile(fileA, []byte("short"), 0o600); err != nil {
		t.Fatalf("write fileA: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("much longer content"), 0o600); err != nil {
		t.Fatalf("write fileB: %v", err)
	}

	equal, err := filesEqual(fileA, fileB)
	if err != nil {
		t.Fatalf("filesEqual: %v", err)
	}
	if equal {
		t.Fatalf("expected files to be different")
	}
}

func TestFilesEqualSameSizeDifferentContent(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")

	if err := os.WriteFile(fileA, []byte("aaaa"), 0o600); err != nil {
		t.Fatalf("write fileA: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("bbbb"), 0o600); err != nil {
		t.Fatalf("write fileB: %v", err)
	}

	equal, err := filesEqual(fileA, fileB)
	if err != nil {
		t.Fatalf("filesEqual: %v", err)
	}
	if equal {
		t.Fatalf("expected files to be different")
	}
}

func TestFilesEqualIdentical(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")

	content := []byte("same content")
	if err := os.WriteFile(fileA, content, 0o600); err != nil {
		t.Fatalf("write fileA: %v", err)
	}
	if err := os.WriteFile(fileB, content, 0o600); err != nil {
		t.Fatalf("write fileB: %v", err)
	}

	equal, err := filesEqual(fileA, fileB)
	if err != nil {
		t.Fatalf("filesEqual: %v", err)
	}
	if !equal {
		t.Fatalf("expected files to be equal")
	}
}

func TestEnsureRegularFileRejectsDirectory(t *testing.T) {
	dir := t.TempDir()

	err := ensureRegularFile(dir)
	if err == nil {
		t.Fatalf("expected error for directory")
	}
	if !errors.Is(err, errExpectedFileIsDir) {
		t.Fatalf("expected errExpectedFileIsDir, got %v", err)
	}
}

func TestCopyFileRejectsSymlinkSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.txt")
	symlink := filepath.Join(dir, "link.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(realFile, []byte("content"), 0o600); err != nil {
		t.Fatalf("write real file: %v", err)
	}
	if err := os.Symlink(realFile, symlink); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	err := copyFile(symlink, dst)
	if err == nil {
		t.Fatalf("expected error copying from symlink")
	}
	if !errors.Is(err, errSymlinkNotAllowed) {
		t.Fatalf("expected errSymlinkNotAllowed, got %v", err)
	}
}

func TestCopyFileRejectsSymlinkDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	realDst := filepath.Join(dir, "real-dst.txt")
	symlinkDst := filepath.Join(dir, "link-dst.txt")

	if err := os.WriteFile(src, []byte("content"), 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(realDst, []byte("old"), 0o600); err != nil {
		t.Fatalf("write real dst: %v", err)
	}
	if err := os.Symlink(realDst, symlinkDst); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	err := copyFile(src, symlinkDst)
	if err == nil {
		t.Fatalf("expected error copying to symlink")
	}
	if !errors.Is(err, errSymlinkNotAllowed) {
		t.Fatalf("expected errSymlinkNotAllowed, got %v", err)
	}
}

func TestListCommandOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	cmd := newListCommand(cfg)
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

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newCurrentCommand(cfg)
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

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}
	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	cmd := newDeleteCommand(cfg)
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

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	cmd := newSwitchCommand(cfg)
	cmd.SetArgs([]string{"work"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("switch command: %v", err)
	}

	status, _ := currentStatus(cfg)
	if status != "work" {
		t.Fatalf("expected work, got %q", status)
	}
}

func TestProfileExistsFunction(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	exists, err := profileExists(cfg, "work")
	if err != nil {
		t.Fatalf("profileExists: %v", err)
	}
	if exists {
		t.Fatalf("expected profile not to exist")
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	exists, err = profileExists(cfg, "work")
	if err != nil {
		t.Fatalf("profileExists after save: %v", err)
	}
	if !exists {
		t.Fatalf("expected profile to exist")
	}
}

func TestCurrentStatusWithDeletedProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}
	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	// Manually delete profile directory (simulating external deletion)
	profileDir, _ := cfg.profileDir("work")
	if err := os.RemoveAll(profileDir); err != nil {
		t.Fatalf("remove profile dir: %v", err)
	}

	// current.json still says "work" but profile doesn't exist
	status, err := currentStatus(cfg)
	if err != nil {
		t.Fatalf("currentStatus: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom> for deleted profile, got %q", status)
	}
}

func TestRestoreRollbackWithExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create target file
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("modified"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	// Create backup
	backup := filepath.Join(dir, "backup.txt")
	if err := os.WriteFile(backup, []byte("original"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	entries := []rollbackEntry{
		{target: target, backup: backup, existed: true},
	}

	if err := restoreRollback(entries); err != nil {
		t.Fatalf("restoreRollback: %v", err)
	}

	// Target should be restored to original content
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("expected 'original', got %q", string(data))
	}
}

func TestRestoreRollbackWithNonExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a file that should be removed during rollback
	target := filepath.Join(dir, "new-file.txt")
	if err := os.WriteFile(target, []byte("new content"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	entries := []rollbackEntry{
		{target: target, existed: false},
	}

	if err := restoreRollback(entries); err != nil {
		t.Fatalf("restoreRollback: %v", err)
	}

	// Target should be removed
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, but it still exists")
	}
}

func TestRestoreRollbackMixedEntries(t *testing.T) {
	dir := t.TempDir()

	// File that existed before (should be restored)
	existingTarget := filepath.Join(dir, "existing.txt")
	existingBackup := filepath.Join(dir, "existing-backup.txt")
	if err := os.WriteFile(existingTarget, []byte("modified"), 0o600); err != nil {
		t.Fatalf("write existing target: %v", err)
	}
	if err := os.WriteFile(existingBackup, []byte("original"), 0o600); err != nil {
		t.Fatalf("write existing backup: %v", err)
	}

	// File that didn't exist before (should be removed)
	newTarget := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(newTarget, []byte("new"), 0o600); err != nil {
		t.Fatalf("write new target: %v", err)
	}

	entries := []rollbackEntry{
		{target: existingTarget, backup: existingBackup, existed: true},
		{target: newTarget, existed: false},
	}

	if err := restoreRollback(entries); err != nil {
		t.Fatalf("restoreRollback: %v", err)
	}

	// Existing file should be restored
	data, err := os.ReadFile(existingTarget)
	if err != nil {
		t.Fatalf("read existing target: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("expected 'original', got %q", string(data))
	}

	// New file should be removed
	if _, err := os.Stat(newTarget); !os.IsNotExist(err) {
		t.Fatalf("expected new file to be removed")
	}
}

func TestRollbackSwitchRestoresProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"v":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Save and switch to work profile
	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}
	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	// Create backup for rollback test
	rollbackDir := t.TempDir()
	backup := filepath.Join(rollbackDir, "settings.json")
	if err := os.WriteFile(backup, []byte(`{"v":1}`), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	entries := []rollbackEntry{
		{target: configPath, backup: backup, existed: true},
	}

	// Modify the config
	if err := os.WriteFile(configPath, []byte(`{"v":2}`), 0o600); err != nil {
		t.Fatalf("modify config: %v", err)
	}

	// Rollback should restore file and profile
	err := rollbackSwitch(cfg, "work", true, entries)
	if err != nil {
		t.Fatalf("rollbackSwitch: %v", err)
	}

	// Config should be restored
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(data) != `{"v":1}` {
		t.Fatalf("expected original config, got %q", string(data))
	}

	// Current profile should be restored
	current, err := readCurrentProfile(cfg)
	if err != nil {
		t.Fatalf("readCurrentProfile: %v", err)
	}
	if current != "work" {
		t.Fatalf("expected 'work', got %q", current)
	}
}

func TestRollbackSwitchWithUnknownPreviousProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()

	// Create tokyo dir for current.json
	tokyoDir, _ := cfg.tokyoDir()
	if err := os.MkdirAll(tokyoDir, 0o700); err != nil {
		t.Fatalf("mkdir tokyo dir: %v", err)
	}

	// Set initial profile
	if err := writeCurrentProfile(cfg, "initial"); err != nil {
		t.Fatalf("writeCurrentProfile: %v", err)
	}

	// Rollback with unknown previous profile (previousProfileKnown = false)
	err := rollbackSwitch(cfg, "", false, nil)
	if err != nil {
		t.Fatalf("rollbackSwitch: %v", err)
	}

	// Current profile should remain unchanged
	current, err := readCurrentProfile(cfg)
	if err != nil {
		t.Fatalf("readCurrentProfile: %v", err)
	}
	if current != "initial" {
		t.Fatalf("expected 'initial', got %q", current)
	}
}

func TestSwitchProfileCreatesConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()

	// Create config file first
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Save profile
	if err := saveProfile(cfg, "work", false); err != nil {
		t.Fatalf("saveProfile: %v", err)
	}

	// Remove the .claude directory
	if err := os.RemoveAll(filepath.Dir(configPath)); err != nil {
		t.Fatalf("remove .claude dir: %v", err)
	}

	// Switch should recreate the directory
	if err := switchProfile(cfg, "work"); err != nil {
		t.Fatalf("switchProfile: %v", err)
	}

	// Config file should exist
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file should exist: %v", err)
	}
}

func TestWriteFileAtomicSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data := []byte(`{"key":"value"}`)
	if err := writeFileAtomic(path, data, 0o600); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("expected %q, got %q", string(data), string(got))
	}

	// Verify permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestWriteFileAtomicCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "test.json")

	data := []byte(`{}`)
	if err := writeFileAtomic(path, data, 0o600); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
}

func TestCopyFileSuccess(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	content := []byte("test content")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", string(content), string(got))
	}
}

func TestStageProfileFilesCleanupOnError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := claudeConfig()

	// Create a profile directory but with missing file
	profileDir, _ := cfg.profileDir("broken")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	// Don't create the settings.json file - this will cause stageProfileFiles to fail

	pairs, err := profilePairs(cfg, profileDir)
	if err != nil {
		t.Fatalf("profilePairs: %v", err)
	}

	// This should fail because the profile file doesn't exist
	_, err = stageProfileFiles(pairs)
	if err == nil {
		t.Fatalf("expected error from stageProfileFiles")
	}

	// Verify no stage files are left behind
	configDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	entries, _ := os.ReadDir(configDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tokyo-stage-") {
			t.Fatalf("stage file not cleaned up: %s", entry.Name())
		}
	}
}
