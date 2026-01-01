package cmd

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

	"github.com/spf13/cobra"
)

type toolConfig struct {
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

func init() {
	rootCmd.AddCommand(newToolCommand(claudeConfig()))
	rootCmd.AddCommand(newToolCommand(codexConfig()))
}

func claudeConfig() toolConfig {
	return toolConfig{
		Name:           "claude",
		DisplayName:    "Claude Code",
		ConfigRelPaths: []string{filepath.Join(".claude", "settings.json")},
	}
}

func codexConfig() toolConfig {
	return toolConfig{
		Name:        "codex",
		DisplayName: "Codex",
		ConfigRelPaths: []string{
			filepath.Join(".codex", "config.toml"),
			filepath.Join(".codex", "auth.json"),
		},
	}
}

func newToolCommand(cfg toolConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cfg.Name,
		Short: fmt.Sprintf("Manage %s configuration profiles", cfg.DisplayName),
	}

	cmd.AddCommand(
		newSwitchCommand(cfg),
		newCurrentCommand(cfg),
		newListCommand(cfg),
		newSaveCommand(cfg),
		newDeleteCommand(cfg),
	)

	return cmd
}

func newSwitchCommand(cfg toolConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <profile>",
		Short: fmt.Sprintf("Switch %s to a profile", cfg.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return switchProfile(cfg, args[0])
		},
	}
}

func newCurrentCommand(cfg toolConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: fmt.Sprintf("Show current %s profile", cfg.DisplayName),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := currentStatus(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), status)
			return nil
		},
	}
}

func newListCommand(cfg toolConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s profiles", cfg.DisplayName),
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := listProfiles(cfg)
			if err != nil {
				return err
			}
			for _, profile := range profiles {
				fmt.Fprintln(cmd.OutOrStdout(), profile)
			}
			return nil
		},
	}
}

func newSaveCommand(cfg toolConfig) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "save <profile>",
		Short: fmt.Sprintf("Save current %s configuration as a profile", cfg.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return saveProfile(cfg, args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing profile")

	return cmd
}

func newDeleteCommand(cfg toolConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <profile>",
		Short: fmt.Sprintf("Delete a %s profile", cfg.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteProfile(cfg, args[0])
		},
	}
}

func (cfg toolConfig) configFiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(cfg.ConfigRelPaths))
	for _, relPath := range cfg.ConfigRelPaths {
		files = append(files, filepath.Join(home, relPath))
	}

	return files, nil
}

func (cfg toolConfig) tokyoDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tokyo", cfg.Name), nil
}

func (cfg toolConfig) profilesDir() (string, error) {
	base, err := cfg.tokyoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "profiles"), nil
}

func (cfg toolConfig) profileDir(profile string) (string, error) {
	profilesDir, err := cfg.profilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(profilesDir, profile), nil
}

func (cfg toolConfig) currentFile() (string, error) {
	base, err := cfg.tokyoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "current.json"), nil
}

func validateProfileName(profile string) error {
	if strings.TrimSpace(profile) == "" {
		return errors.New("profile name cannot be empty")
	}
	if filepath.Base(profile) != profile || strings.Contains(profile, string(os.PathSeparator)) {
		return fmt.Errorf("invalid profile name: %q", profile)
	}
	return nil
}

func listProfiles(cfg toolConfig) ([]string, error) {
	profilesDir, err := cfg.profilesDir()
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

func saveProfile(cfg toolConfig, profile string, force bool) error {
	if err := validateProfileName(profile); err != nil {
		return err
	}

	profileDir, err := cfg.profileDir(profile)
	if err != nil {
		return err
	}

	if _, err := os.Stat(profileDir); err == nil {
		if !force {
			return fmt.Errorf("profile %q already exists (use --force to overwrite)", profile)
		}
		if err := os.RemoveAll(profileDir); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}

	configFiles, err := cfg.configFiles()
	if err != nil {
		return err
	}

	for _, src := range configFiles {
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("config file not found: %s", src)
			}
			return err
		}
		dst := filepath.Join(profileDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}

	return nil
}

func deleteProfile(cfg toolConfig, profile string) error {
	if err := validateProfileName(profile); err != nil {
		return err
	}

	profileDir, err := cfg.profileDir(profile)
	if err != nil {
		return err
	}

	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile %q not found", profile)
		}
		return err
	}

	return os.RemoveAll(profileDir)
}

func currentStatus(cfg toolConfig) (string, error) {
	profile, err := readCurrentProfile(cfg)
	if err != nil {
		return "", err
	}
	if profile == "" {
		return "<custom>", nil
	}

	exists, err := profileExists(cfg, profile)
	if err != nil {
		return "", err
	}
	if !exists {
		return "<custom>", nil
	}

	match, err := profileMatches(cfg, profile)
	if err != nil {
		return "", err
	}
	if match {
		return profile, nil
	}
	return fmt.Sprintf("%s (modified)", profile), nil
}

func switchProfile(cfg toolConfig, profile string) error {
	if err := validateProfileName(profile); err != nil {
		return err
	}

	profileDir, err := cfg.profileDir(profile)
	if err != nil {
		return err
	}
	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile %q not found", profile)
		}
		return err
	}

	pairs, err := profilePairs(cfg, profileDir)
	if err != nil {
		return err
	}

	if err := ensureProfileFilesExist(pairs); err != nil {
		return err
	}

	stageFiles, err := stageProfileFiles(pairs)
	if err != nil {
		return err
	}
	defer cleanupStageFiles(stageFiles)

	rollbackDir, err := createRollbackDir(cfg)
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
			rollbackErr := restoreRollback(rollbackEntries)
			if rollbackErr != nil {
				return fmt.Errorf("switch failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return err
		}
		delete(stageFiles, pair.dst)
	}

	if err := writeCurrentProfile(cfg, profile); err != nil {
		rollbackErr := restoreRollback(rollbackEntries)
		if rollbackErr != nil {
			return fmt.Errorf("switch failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	return nil
}

func profileExists(cfg toolConfig, profile string) (bool, error) {
	profileDir, err := cfg.profileDir(profile)
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

func profileMatches(cfg toolConfig, profile string) (bool, error) {
	profileDir, err := cfg.profileDir(profile)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(profileDir); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	pairs, err := profilePairs(cfg, profileDir)
	if err != nil {
		return false, err
	}

	for _, pair := range pairs {
		if _, err := os.Stat(pair.src); err != nil {
			if os.IsNotExist(err) {
				return false, fmt.Errorf("profile is missing file: %s", filepath.Base(pair.src))
			}
			return false, err
		}
		if _, err := os.Stat(pair.dst); err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
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

func profilePairs(cfg toolConfig, profileDir string) ([]filePair, error) {
	configFiles, err := cfg.configFiles()
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

func ensureProfileFilesExist(pairs []filePair) error {
	for _, pair := range pairs {
		if _, err := os.Stat(pair.src); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("profile is missing file: %s", filepath.Base(pair.src))
			}
			return err
		}
	}
	return nil
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

func createRollbackDir(cfg toolConfig) (string, error) {
	base, err := cfg.tokyoDir()
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
		if _, err := os.Stat(pair.dst); err != nil {
			if os.IsNotExist(err) {
				entries = append(entries, rollbackEntry{target: pair.dst, existed: false})
				continue
			}
			return nil, err
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
	var firstErr error
	for _, entry := range entries {
		if entry.existed {
			if err := copyFile(entry.backup, entry.target); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := os.Remove(entry.target); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func readCurrentProfile(cfg toolConfig) (string, error) {
	currentFile, err := cfg.currentFile()
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

func writeCurrentProfile(cfg toolConfig, profile string) error {
	currentFile, err := cfg.currentFile()
	if err != nil {
		return err
	}

	if err := ensureParentDir(currentFile); err != nil {
		return err
	}

	state := currentState{Profile: profile}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(currentFile, data, 0o600)
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

func copyFile(src, dst string) error {
	if err := ensureParentDir(dst); err != nil {
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
