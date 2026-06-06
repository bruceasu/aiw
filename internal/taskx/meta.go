package taskx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/fsx"
)

const (
	OpenspecDir   = "openspec"
	ChangesDir    = "openspec/changes"
	SpecsDir      = "openspec/specs"
	ArchiveDir    = "openspec/archive"
	RegistryFile  = "openspec/registry.json"
	WorktreeDir   = ".wt"
	GitignoreFile = ".gitignore"
)

type TaskMeta struct {
	ID       string
	Type     string
	Status   string
	Created  string
	Updated  string
	Branch   string
	Worktree string
	Specs    []string
	Tags     []string
}

type RegistryEntry struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Branch    string `json:"branch"`
	Worktree  string `json:"worktree"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
}

func Today() string {
	return time.Now().Format("2006-01-02")
}

func TaskDir(id string) string {
	return filepath.Join(ChangesDir, id)
}

func TaskMetaPath(id string) string {
	return filepath.Join(TaskDir(id), "task.toml")
}

func ReadTaskMeta(path string) (TaskMeta, error) {
	file, err := os.Open(path)
	if err != nil {
		return TaskMeta{}, err
	}
	defer file.Close()

	meta := TaskMeta{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		switch key {
		case "id":
			meta.ID = value
		case "type":
			meta.Type = value
		case "status":
			meta.Status = value
		case "created":
			meta.Created = value
		case "updated":
			meta.Updated = value
		case "branch":
			meta.Branch = value
		case "worktree":
			meta.Worktree = value
		}
	}
	return meta, scanner.Err()
}

func WriteTaskMeta(path string, meta TaskMeta) error {
	content := fmt.Sprintf(`id = "%s"
type = "%s"
status = "%s"
created = "%s"
updated = "%s"
branch = "%s"
worktree = "%s"
`,
		meta.ID,
		meta.Type,
		meta.Status,
		meta.Created,
		meta.Updated,
		meta.Branch,
		meta.Worktree,
	)
	return os.WriteFile(path, []byte(content), 0o644)
}

func EnsureWorktreeIgnored() error {
	entry := WorktreeDir + "/"
	if !fsx.Exists(GitignoreFile) {
		if err := os.WriteFile(GitignoreFile, []byte(entry+"\n"), 0o644); err != nil {
			return err
		}
		fmt.Println("created:", GitignoreFile)
		return nil
	}
	b, err := os.ReadFile(GitignoreFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == WorktreeDir || trimmed == entry {
			fmt.Println("exists:", GitignoreFile, entry)
			return nil
		}
	}
	content := string(b)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"
	if err := os.WriteFile(GitignoreFile, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Println("updated:", GitignoreFile, entry)
	return nil
}
