package profile

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrSymlinkNotAllowed   = errors.New("symlink not allowed")
	ErrExpectedFileIsDir   = errors.New("expected file but found directory")
	ErrExpectedRegularFile = errors.New("expected regular file")

	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrConfigFileNotFound   = errors.New("config file not found")
	ErrProfileMissingFile   = errors.New("profile is missing file")
)

type userError struct {
	kind error
	msg  string
}

func (e *userError) Error() string {
	return e.msg
}

func (e *userError) Unwrap() error {
	return e.kind
}

func newUserError(kind error, msg string) error {
	return &userError{kind: kind, msg: msg}
}

type Tool struct {
	Name           string
	DisplayName    string
	ConfigRelPaths []string
}

type currentState struct {
	Profile string `json:"profile"`
}

type filePair struct {
	src string
	dst string
}

type rollbackEntry struct {
	target  string
	backup  string
	existed bool
}

func ClaudeTool() Tool {
	return Tool{
		Name:           "claude",
		DisplayName:    "Claude Code",
		ConfigRelPaths: []string{filepath.Join(".claude", "settings.json")},
	}
}

func CodexTool() Tool {
	return Tool{
		Name:        "codex",
		DisplayName: "Codex",
		ConfigRelPaths: []string{
			filepath.Join(".codex", "config.toml"),
			filepath.Join(".codex", "auth.json"),
		},
	}
}

func (t Tool) configFiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(t.ConfigRelPaths))
	for _, relPath := range t.ConfigRelPaths {
		files = append(files, filepath.Join(home, relPath))
	}

	return files, nil
}

func (t Tool) tokyoDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tokyo", t.Name), nil
}

func (t Tool) profilesDir() (string, error) {
	base, err := t.tokyoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "profiles"), nil
}

func (t Tool) profileDir(profile string) (string, error) {
	profilesDir, err := t.profilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(profilesDir, profile), nil
}

func (t Tool) currentFile() (string, error) {
	base, err := t.tokyoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "current.json"), nil
}

func ValidateProfileName(profile string) error {
	const maxLen = 64

	if strings.TrimSpace(profile) == "" {
		return errors.New("profile name cannot be empty")
	}
	if strings.TrimSpace(profile) != profile {
		return errors.New("profile name cannot start or end with whitespace")
	}
	if len(profile) > maxLen {
		return fmt.Errorf("profile name too long (max %d characters)", maxLen)
	}
	if profile == "<custom>" {
		return errors.New("profile name is reserved")
	}
	if strings.HasSuffix(profile, " (modified)") {
		return errors.New("profile name cannot end with ' (modified)'")
	}
	if strings.HasPrefix(profile, ".") {
		return errors.New("profile name cannot start with '.'")
	}
	if filepath.Base(profile) != profile || strings.Contains(profile, string(os.PathSeparator)) {
		return fmt.Errorf("invalid profile name: %q", profile)
	}

	for _, r := range profile {
		if r > 0x7f {
			return fmt.Errorf("invalid profile name: %q (ASCII only)", profile)
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("invalid profile name: %q (allowed: A-Z a-z 0-9 _ -)", profile)
	}

	return nil
}

func List(t Tool) ([]string, error) {
	profilesDir, err := t.profilesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			profiles = append(profiles, entry.Name())
		}
	}

	sort.Strings(profiles)

	return profiles, nil
}

func Save(t Tool, profile string, force bool) error {
	if err := ValidateProfileName(profile); err != nil {
		return err
	}

	profileDir, err := t.profileDir(profile)
	if err != nil {
		return err
	}

	if force {
		if err := os.RemoveAll(profileDir); err != nil {
			return err
		}
		if err := os.MkdirAll(profileDir, 0o700); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(profileDir), 0o700); err != nil {
			return err
		}
		if err := os.Mkdir(profileDir, 0o700); err != nil {
			if os.IsExist(err) {
				return newUserError(ErrProfileAlreadyExists, fmt.Sprintf("profile %q already exists (use --force to overwrite)", profile))
			}
			return err
		}
	}

	configFiles, err := t.configFiles()
	if err != nil {
		return err
	}

	for _, src := range configFiles {
		dst := filepath.Join(profileDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			if os.IsNotExist(err) {
				return newUserError(ErrConfigFileNotFound, fmt.Sprintf("config file not found: %s", src))
			}
			return err
		}
	}

	return nil
}

func Delete(t Tool, profile string) (cleared bool, err error) {
	if err := ValidateProfileName(profile); err != nil {
		return false, err
	}

	profileDir, err := t.profileDir(profile)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return false, newUserError(ErrProfileNotFound, fmt.Sprintf("profile %q not found", profile))
		}
		return false, err
	}

	current, err := readCurrentProfile(t)
	if err != nil {
		return false, err
	}
	wasCurrent := current == profile

	if err := os.RemoveAll(profileDir); err != nil {
		return false, err
	}

	if wasCurrent {
		if err := writeCurrentProfile(t, ""); err != nil {
			return false, err
		}
	}

	return wasCurrent, nil
}

func Current(t Tool) (string, error) {
	profile, err := readCurrentProfile(t)
	if err != nil {
		return "", err
	}
	if profile == "" {
		return "<custom>", nil
	}

	exists, err := Exists(t, profile)
	if err != nil {
		return "", err
	}
	if !exists {
		return "<custom>", nil
	}

	match, err := matches(t, profile)
	if err != nil {
		return "", err
	}
	if match {
		return profile, nil
	}
	return fmt.Sprintf("%s (modified)", profile), nil
}

func Switch(t Tool, profile string) error {
	if err := ValidateProfileName(profile); err != nil {
		return err
	}

	previousProfile := ""
	previousProfileKnown := false
	if current, err := readCurrentProfile(t); err == nil {
		previousProfile = current
		previousProfileKnown = true
	}

	profileDir, err := t.profileDir(profile)
	if err != nil {
		return err
	}
	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return newUserError(ErrProfileNotFound, fmt.Sprintf("profile %q not found", profile))
		}
		return err
	}

	pairs, err := profilePairs(t, profileDir)
	if err != nil {
		return err
	}

	stageFiles, err := stageProfileFiles(pairs)
	if err != nil {
		return err
	}
	defer cleanupStageFiles(stageFiles)

	rollbackDir, err := createRollbackDir(t)
	if err != nil {
		return err
	}
	defer os.RemoveAll(rollbackDir)

	rollbackEntries, err := backupCurrentFiles(pairs, rollbackDir)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		stagePath := stageFiles[pair.dst]
		if err := os.Rename(stagePath, pair.dst); err != nil {
			rollbackErr := rollbackSwitch(t, previousProfile, previousProfileKnown, rollbackEntries)
			if rollbackErr != nil {
				return errors.Join(fmt.Errorf("switch failed: %w", err), rollbackErr)
			}
			return fmt.Errorf("switch failed: %w", err)
		}
		delete(stageFiles, pair.dst)
	}

	if err := writeCurrentProfile(t, profile); err != nil {
		rollbackErr := rollbackSwitch(t, previousProfile, previousProfileKnown, rollbackEntries)
		if rollbackErr != nil {
			return errors.Join(fmt.Errorf("switch failed: %w", err), rollbackErr)
		}
		return fmt.Errorf("switch failed: %w", err)
	}

	return nil
}

func Exists(t Tool, profile string) (bool, error) {
	profileDir, err := t.profileDir(profile)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func matches(t Tool, profile string) (bool, error) {
	profileDir, err := t.profileDir(profile)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	pairs, err := profilePairs(t, profileDir)
	if err != nil {
		return false, err
	}

	for _, pair := range pairs {
		if err := ensureRegularFile(pair.src); err != nil {
			if os.IsNotExist(err) {
				return false, newUserError(ErrProfileMissingFile, fmt.Sprintf("profile is missing file: %s", filepath.Base(pair.src)))
			}
			return false, err
		}
		exists, err := ensureRegularFileIfExists(pair.dst)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
		same, err := filesEqual(pair.src, pair.dst)
		if err != nil {
			return false, err
		}
		if !same {
			return false, nil
		}
	}

	return true, nil
}

func profilePairs(t Tool, profileDir string) ([]filePair, error) {
	configFiles, err := t.configFiles()
	if err != nil {
		return nil, err
	}

	pairs := make([]filePair, 0, len(configFiles))
	for _, dst := range configFiles {
		src := filepath.Join(profileDir, filepath.Base(dst))
		pairs = append(pairs, filePair{src: src, dst: dst})
	}

	return pairs, nil
}

func stageProfileFiles(pairs []filePair) (map[string]string, error) {
	stageFiles := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		if err := ensureParentDir(pair.dst); err != nil {
			cleanupStageFiles(stageFiles)
			return nil, err
		}
		tmpFile, err := os.CreateTemp(filepath.Dir(pair.dst), ".tokyo-stage-")
		if err != nil {
			cleanupStageFiles(stageFiles)
			return nil, err
		}
		if err := copyFileToFile(pair.src, tmpFile); err != nil {
			os.Remove(tmpFile.Name())
			cleanupStageFiles(stageFiles)
			if os.IsNotExist(err) {
				return nil, newUserError(ErrProfileMissingFile, fmt.Sprintf("profile is missing file: %s", filepath.Base(pair.src)))
			}
			return nil, err
		}
		stageFiles[pair.dst] = tmpFile.Name()
	}
	return stageFiles, nil
}

func cleanupStageFiles(stageFiles map[string]string) {
	for _, path := range stageFiles {
		_ = os.Remove(path)
	}
}

func createRollbackDir(t Tool) (string, error) {
	base, err := t.tokyoDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "rollback-")
}

func backupCurrentFiles(pairs []filePair, rollbackDir string) ([]rollbackEntry, error) {
	entries := make([]rollbackEntry, 0, len(pairs))
	for _, pair := range pairs {
		existed, err := ensureRegularFileIfExists(pair.dst)
		if err != nil {
			return nil, err
		}
		if !existed {
			entries = append(entries, rollbackEntry{target: pair.dst, existed: false})
			continue
		}
		backup := filepath.Join(rollbackDir, filepath.Base(pair.dst))
		if err := copyFile(pair.dst, backup); err != nil {
			return nil, err
		}
		entries = append(entries, rollbackEntry{target: pair.dst, backup: backup, existed: true})
	}
	return entries, nil
}

func restoreRollback(entries []rollbackEntry) error {
	var errs []error
	for _, entry := range entries {
		if entry.existed {
			if err := copyFile(entry.backup, entry.target); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.Remove(entry.target); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func rollbackSwitch(t Tool, previousProfile string, previousProfileKnown bool, entries []rollbackEntry) error {
	var errs []error
	if err := restoreRollback(entries); err != nil {
		errs = append(errs, err)
	}
	if previousProfileKnown {
		if err := writeCurrentProfile(t, previousProfile); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func readCurrentProfile(t Tool) (string, error) {
	currentFile, err := t.currentFile()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(currentFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var state currentState
	if err := json.Unmarshal(data, &state); err != nil {
		return "", err
	}
	return state.Profile, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	if err := rejectNonRegularFile(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tokyo-")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(perm); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	if err := ensureRegularFile(path); err != nil {
		os.Remove(path)
		return fmt.Errorf("post-rename validation failed: %w", err)
	}
	return nil
}

func writeCurrentProfile(t Tool, profile string) error {
	currentFile, err := t.currentFile()
	if err != nil {
		return err
	}

	state := currentState{Profile: profile}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return writeFileAtomic(currentFile, data, 0o600)
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

func ensureRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", ErrSymlinkNotAllowed, path)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %s", ErrExpectedFileIsDir, path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s", ErrExpectedRegularFile, path)
	}
	return nil
}

func ensureRegularFileIfExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true, fmt.Errorf("%w: %s", ErrSymlinkNotAllowed, path)
	}
	if info.IsDir() {
		return true, fmt.Errorf("%w: %s", ErrExpectedFileIsDir, path)
	}
	if !info.Mode().IsRegular() {
		return true, fmt.Errorf("%w: %s", ErrExpectedRegularFile, path)
	}
	return true, nil
}

func rejectNonRegularFile(path string) error {
	_, err := ensureRegularFileIfExists(path)
	return err
}

func copyFile(src, dst string) error {
	if err := ensureRegularFile(src); err != nil {
		return err
	}
	if err := ensureParentDir(dst); err != nil {
		return err
	}
	if err := rejectNonRegularFile(dst); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyFileToFile(src string, dst *os.File) error {
	if err := ensureRegularFile(src); err != nil {
		dst.Close()
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		dst.Close()
		return err
	}
	defer in.Close()

	if _, err := io.Copy(dst, in); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Sync(); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

func filesEqual(pathA, pathB string) (bool, error) {
	if err := ensureRegularFile(pathA); err != nil {
		return false, err
	}
	if err := ensureRegularFile(pathB); err != nil {
		return false, err
	}

	infoA, err := os.Stat(pathA)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(pathB)
	if err != nil {
		return false, err
	}
	if infoA.Size() != infoB.Size() {
		return false, nil
	}

	hashA, err := fileHash(pathA)
	if err != nil {
		return false, err
	}
	hashB, err := fileHash(pathB)
	if err != nil {
		return false, err
	}
	return hashA == hashB, nil
}

func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
