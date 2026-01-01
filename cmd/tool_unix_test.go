//go:build !windows

package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteCurrentProfileRejectsNonRegularPaths(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T, path, home string)
		wantErr error
	}{
		{
			name: "normal_write",
			setup: func(t *testing.T, path, home string) {
				// No setup - test normal write to non-existent file
			},
			wantErr: nil,
		},
		{
			name: "symlink",
			setup: func(t *testing.T, path, home string) {
				target := filepath.Join(home, "target.json")
				if err := os.WriteFile(target, []byte(`{"x":1}`), 0o600); err != nil {
					t.Fatalf("write target: %v", err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatalf("symlink: %v", err)
				}
			},
			wantErr: errSymlinkNotAllowed,
		},
		{
			name: "directory",
			setup: func(t *testing.T, path, home string) {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
			},
			wantErr: errExpectedFileIsDir,
		},
		{
			name: "fifo",
			setup: func(t *testing.T, path, home string) {
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Fatalf("mkfifo: %v", err)
				}
			},
			wantErr: errExpectedRegularFile,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			cfg := claudeConfig()
			currentFile, err := cfg.currentFile()
			if err != nil {
				t.Fatalf("currentFile: %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(currentFile), 0o700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			tc.setup(t, currentFile, home)

			err = writeCurrentProfile(cfg, "work")
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
			} else {
				if err == nil || !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
			}
		})
	}
}
