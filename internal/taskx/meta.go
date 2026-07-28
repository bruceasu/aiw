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
	OpenspecDir        = "openspec"
	ChangesDir         = "openspec/changes"
	SpecsDir           = "openspec/specs"
	ArchiveDir         = "openspec/changes/archive"
	LegacyArchiveDir   = "openspec/archive"
	RegistryFile       = "openspec/registry.json"
	WorktreeDir        = ".wt"
	GitignoreFile      = ".gitignore"
	TaskMetaFile       = "task.toml"
	LegacyTaskMetaFile = "tasks.toml"
)

type TaskMeta struct {
	ID       string
	Type     string
	Status   string
	Created  string
	Updated  string
	Branch   string
	Worktree string
	Session  string
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

func ArchiveTaskDir(name string) string {
	return filepath.Join(ArchiveDir, name)
}

func TaskMetaPath(id string) string {
	return filepath.Join(TaskDir(id), TaskMetaFile)
}

func ResolveTaskMetaPath(id string) string {
	dir := TaskDir(id)
	primary := filepath.Join(dir, TaskMetaFile)
	if fsx.Exists(primary) {
		return primary
	}
	legacy := filepath.Join(dir, LegacyTaskMetaFile)
	if fsx.Exists(legacy) {
		return legacy
	}
	return primary
}

func ResolveTaskMetaPathInDir(dir string) string {
	primary := filepath.Join(dir, TaskMetaFile)
	if fsx.Exists(primary) {
		return primary
	}
	legacy := filepath.Join(dir, LegacyTaskMetaFile)
	if fsx.Exists(legacy) {
		return legacy
	}
	return primary
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
		case "session", "session_id":
			meta.Session = value
		case "specs":
			meta.Specs = parseStringArray(parts[1])
		case "tags":
			meta.Tags = parseStringArray(parts[1])
		}
	}
	return meta, scanner.Err()
}

func parseStringArray(raw string) []string {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		item = strings.Trim(item, `"`)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func WriteTaskMeta(path string, meta TaskMeta) error {
	var specsLine string
	if len(meta.Specs) > 0 {
		specsLine = fmt.Sprintf("specs = [%s]\n", quoteStringArray(meta.Specs))
	}
	var tagsLine string
	if len(meta.Tags) > 0 {
		tagsLine = fmt.Sprintf("tags = [%s]\n", quoteStringArray(meta.Tags))
	}

	content := fmt.Sprintf(`id = "%s"
type = "%s"
status = "%s"
created = "%s"
updated = "%s"
branch = "%s"
worktree = "%s"
session = "%s"
%s%s`,
		meta.ID,
		meta.Type,
		meta.Status,
		meta.Created,
		meta.Updated,
		meta.Branch,
		meta.Worktree,
		meta.Session,
		specsLine,
		tagsLine,
	)
	return os.WriteFile(path, []byte(content), 0o644)
}

func quoteStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		escaped := strings.ReplaceAll(value, `"`, `\"`)
		quoted = append(quoted, fmt.Sprintf(`"%s"`, escaped))
	}
	return strings.Join(quoted, ", ")
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
