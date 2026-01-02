package profile

import (
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
		{name: "leading_trailing_whitespace", profile: " work ", wantErr: true},
		{name: "dotfile", profile: ".work", wantErr: true},
		{name: "path", profile: "a/b", wantErr: true},
		{name: "internal_space", profile: "my profile", wantErr: true},
		{name: "reserved_custom", profile: "<custom>", wantErr: true},
		{name: "modified_suffix", profile: "work (modified)", wantErr: true},
		{name: "ok", profile: "work", wantErr: false},
		{name: "ok_with_hyphen", profile: "work-1", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProfileName(tc.profile)
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

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	status, err := Current(tool)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom>, got %q", status)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Switch(tool, "work"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	status, err = Current(tool)
	if err != nil {
		t.Fatalf("Current after switch: %v", err)
	}
	if status != "work" {
		t.Fatalf("expected work, got %q", status)
	}

	if err := os.WriteFile(configPath, []byte(`{"x":2}`), 0o600); err != nil {
		t.Fatalf("write config (modified): %v", err)
	}

	status, err = Current(tool)
	if err != nil {
		t.Fatalf("Current after modify: %v", err)
	}
	if status != "work (modified)" {
		t.Fatalf("expected work (modified), got %q", status)
	}

	cleared, err := Delete(tool, "work")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !cleared {
		t.Fatalf("expected deleting active profile to clear current")
	}

	status, err = Current(tool)
	if err != nil {
		t.Fatalf("Current after delete: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom>, got %q", status)
	}
}

func TestSaveProfileWithoutForceFailsIfExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := Save(tool, "work", false); err == nil {
		t.Fatalf("expected error on second save without --force, got nil")
	}
	if err := Save(tool, "work", true); err != nil {
		t.Fatalf("Save with --force: %v", err)
	}
}

func TestCurrentStatusRejectsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Switch(tool, "work"); err != nil {
		t.Fatalf("Switch: %v", err)
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

	if _, err := Current(tool); err == nil || !errors.Is(err, ErrSymlinkNotAllowed) {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestCodexLifecycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := CodexTool()
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

	status, err := Current(tool)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom>, got %q", status)
	}

	if err := Save(tool, "personal", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Switch(tool, "personal"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	status, err = Current(tool)
	if err != nil {
		t.Fatalf("Current after switch: %v", err)
	}
	if status != "personal" {
		t.Fatalf("expected personal, got %q", status)
	}

	if err := os.WriteFile(configPath, []byte(`key = "value2"`), 0o600); err != nil {
		t.Fatalf("write config.toml (modified): %v", err)
	}

	status, err = Current(tool)
	if err != nil {
		t.Fatalf("Current after modify: %v", err)
	}
	if status != "personal (modified)" {
		t.Fatalf("expected personal (modified), got %q", status)
	}

	if err := Switch(tool, "personal"); err != nil {
		t.Fatalf("Switch again: %v", err)
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

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	profiles, err := List(tool)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected empty list, got %v", profiles)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save work: %v", err)
	}
	if err := Save(tool, "personal", false); err != nil {
		t.Fatalf("Save personal: %v", err)
	}
	if err := Save(tool, "alpha", false); err != nil {
		t.Fatalf("Save alpha: %v", err)
	}

	profiles, err = List(tool)
	if err != nil {
		t.Fatalf("List after save: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(profiles))
	}
	if profiles[0] != "alpha" || profiles[1] != "personal" || profiles[2] != "work" {
		t.Fatalf("expected sorted [alpha personal work], got %v", profiles)
	}
}

func TestDeleteNonExistentProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()

	_, err := Delete(tool, "nonexistent")
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

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save work: %v", err)
	}
	if err := Save(tool, "personal", false); err != nil {
		t.Fatalf("Save personal: %v", err)
	}
	if err := Switch(tool, "work"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	cleared, err := Delete(tool, "personal")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if cleared {
		t.Fatalf("expected cleared=false for non-active profile")
	}

	status, err := Current(tool)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if status != "work" {
		t.Fatalf("expected work, got %q", status)
	}
}

func TestSwitchToNonExistentProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()

	err := Switch(tool, "nonexistent")
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

	tool := ClaudeTool()

	err := Save(tool, "work", false)
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

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	profilesDir := filepath.Join(home, ".config", "tokyo", "claude", "profiles", "work")
	profileFile := filepath.Join(profilesDir, "settings.json")
	if err := os.Remove(profileFile); err != nil {
		t.Fatalf("remove profile file: %v", err)
	}

	err := Switch(tool, "work")
	if err == nil {
		t.Fatalf("expected error switching with missing profile file")
	}
	if !strings.Contains(err.Error(), "missing file") {
		t.Fatalf("expected 'missing file' error, got %v", err)
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
	if !errors.Is(err, ErrExpectedFileIsDir) {
		t.Fatalf("expected ErrExpectedFileIsDir, got %v", err)
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
	if !errors.Is(err, ErrSymlinkNotAllowed) {
		t.Fatalf("expected ErrSymlinkNotAllowed, got %v", err)
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
	if !errors.Is(err, ErrSymlinkNotAllowed) {
		t.Fatalf("expected ErrSymlinkNotAllowed, got %v", err)
	}
}

func TestProfileExistsFunction(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	exists, err := Exists(tool, "work")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatalf("expected profile not to exist")
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err = Exists(tool, "work")
	if err != nil {
		t.Fatalf("Exists after save: %v", err)
	}
	if !exists {
		t.Fatalf("expected profile to exist")
	}
}

func TestCurrentStatusWithDeletedProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()
	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Switch(tool, "work"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	profileDir := filepath.Join(home, ".config", "tokyo", "claude", "profiles", "work")
	if err := os.RemoveAll(profileDir); err != nil {
		t.Fatalf("remove profile dir: %v", err)
	}

	status, err := Current(tool)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if status != "<custom>" {
		t.Fatalf("expected <custom> for deleted profile, got %q", status)
	}
}

func TestSwitchProfileCreatesConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tool := ClaudeTool()

	configPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Save(tool, "work", false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := os.RemoveAll(filepath.Dir(configPath)); err != nil {
		t.Fatalf("remove .claude dir: %v", err)
	}

	if err := Switch(tool, "work"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file should exist: %v", err)
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
