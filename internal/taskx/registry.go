package taskx

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func WriteRegistry() error {
	entries, err := os.ReadDir(ChangesDir)
	if err != nil {
		return err
	}
	var changes []RegistryEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := ReadTaskMeta(filepath.Join(ChangesDir, e.Name(), "task.toml"))
		if err != nil {
			continue
		}
		changes = append(changes, RegistryEntry{
			ID:        meta.ID,
			Status:    meta.Status,
			Branch:    meta.Branch,
			Worktree:  meta.Worktree,
			Path:      filepath.ToSlash(filepath.Join(ChangesDir, e.Name())),
			UpdatedAt: meta.Updated,
		})
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].ID < changes[j].ID
	})
	payload := map[string]any{
		"version": "1",
		"updated": time.Now().Format(time.RFC3339),
		"changes": changes,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	lockPath := RegistryFile + ".lock"
	unlock, err := acquireFileLock(lockPath, 2*time.Second)
	if err != nil {
		return err
	}
	defer unlock()
	return atomicWriteFile(RegistryFile, b, 0o644)
}

func acquireFileLock(lockPath string, timeout time.Duration) (func(), error) {
	deadline := time.Now().Add(timeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			return func() {
				_ = f.Close()
				_ = os.Remove(lockPath)
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, errors.New("timeout acquiring registry lock")
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".registry-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err == nil {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpPath, path)
}
