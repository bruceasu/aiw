package issue

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"aiw/internal/repo"
)

var requirementSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var numberedRequirement = regexp.MustCompile(`(?i)^REQ([0-9]+)-.+$`)
var errSequenceCorrupt = errors.New("Requirement sequence is corrupt")

func runtimeRoot() string { return filepath.Join(repo.Root(), ".ai", "requirements") }

// ValidSlug is deliberately narrower than ValidID; historical IDs stay valid.
func ValidSlug(slug string) bool { return requirementSlug.MatchString(slug) }

// CreateNumbered allocates only when the host actually creates a Requirement.
func CreateNumbered(slug, title string) (Meta, error) {
	if !ValidSlug(slug) {
		return Meta{}, errors.New("requirement slug must contain lowercase letters or digits separated by single hyphens")
	}
	return createWithSequence(slug, title, true)
}

func createWithSequence(id, title string, automatic bool) (Meta, error) {
	if !ValidID(id) || (!automatic && (id == "." || id == ".." || strings.EqualFold(id, archiveRoot) || strings.EqualFold(id, cancelledRoot))) {
		return Meta{}, errors.New("invalid or reserved requirement id")
	}
	root := repo.Root()
	storage := Root
	if automatic || numberedIssue.MatchString(id) { storage = IssueRoot }
	runtime := runtimeRoot()
	if storage == IssueRoot { runtime = filepath.Join(root, ".ai", "issues") }
	sequence := filepath.Join(runtime, "sequence")
	if err := os.MkdirAll(filepath.Dir(sequence), 0o755); err != nil { return Meta{}, err }
	lock, err := os.OpenFile(sequence+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil { return Meta{}, fmt.Errorf("acquire Requirement creation lock (do not remove an active lock): %w", err) }
	defer func() { _ = lock.Close(); _ = os.Remove(sequence+".lock") }()

	var number uint64
	numbered := automatic
	if !automatic {
		if storage == IssueRoot { number, numbered, err = issueNumber(id) } else { number, numbered, err = requirementNumber(id) }
		if err != nil { return Meta{}, err }
	}
	if numbered {
		if storage == IssueRoot {
			number, err = reserveRecordNumber(sequence, root, number, IssueRoot, issueNumber)
		} else { number, err = reserveRequirementNumber(sequence, root, number) }
		if err != nil { return Meta{}, err }
		if automatic { id = fmt.Sprintf("ISSUE-%03d", number) }
	}
	return createExact(id, title)
}

func requirementNumber(id string) (uint64, bool, error) {
	match := numberedRequirement.FindStringSubmatch(id)
	if match == nil { return 0, false, nil }
	number, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil || number == 0 { return 0, true, fmt.Errorf("invalid Requirement number in %q", id) }
	return number, true, nil
}

func issueNumber(id string) (uint64, bool, error) {
	match := numberedIssue.FindStringSubmatch(id)
	if match == nil { return 0, false, nil }
	number, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil || number == 0 { return 0, true, fmt.Errorf("invalid Issue number in %q", id) }
	return number, true, nil
}

// Only missing or malformed records permit recovery; I/O failures do not.
func readRequirementSequence(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil { return 0, err }
	defer file.Close()
	info, err := file.Stat()
	if err != nil { return 0, err }
	if !info.Mode().IsRegular() { return 0, errors.New("Requirement sequence is not a regular file") }
	if info.Size() == 0 { return 0, fmt.Errorf("%w: empty file", errSequenceCorrupt) }
	last := make([]byte, 1)
	if _, err := file.ReadAt(last, info.Size()-1); err != nil { return 0, err }
	if last[0] != '\n' { return 0, fmt.Errorf("%w: incomplete record", errSequenceCorrupt) }
	var highest uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		number, err := strconv.ParseUint(scanner.Text(), 10, 64)
		if errors.Is(err, strconv.ErrRange) { return 0, errors.New("Requirement sequence number overflows uint64") }
		if err != nil || number <= highest { return 0, errSequenceCorrupt }
		highest = number
	}
	return highest, scanner.Err()
}

func reserveRequirementNumber(sequence, root string, requested uint64) (uint64, error) {
	return reserveRecordNumber(sequence, root, requested, Root, requirementNumber)
}

func reserveRecordNumber(sequence, root string, requested uint64, storage string, parseNumber func(string) (uint64, bool, error)) (uint64, error) {
	highest, sequenceErr := readRequirementSequence(sequence)
	recovering := errors.Is(sequenceErr, os.ErrNotExist) || errors.Is(sequenceErr, errSequenceCorrupt)
	if sequenceErr != nil && !recovering { return 0, sequenceErr }
	var directoryHighest uint64
	seen := make(map[string]bool)
	for _, base := range []string{filepath.Join(root, storage), storage} {
		for _, suffix := range []string{"", archiveRoot, cancelledRoot} {
			path, err := filepath.Abs(filepath.Join(base, suffix))
			if err != nil { return 0, err }
			if seen[path] { continue }
			seen[path] = true
			entries, err := readRequirementNumberingDirectory(path)
			if err != nil { return 0, err }
			for _, entry := range entries {
				if !entry.IsDir() { continue }
				number, _, err := parseNumber(entry.Name())
				if err != nil { return 0, err }
				if number > highest { highest = number }
				if number > directoryHighest { directoryHighest = number }
			}
		}
	}
	if requested == 0 {
		if highest == ^uint64(0) { return 0, errors.New("Requirement sequence is exhausted") }
		requested = highest + 1
	} else if requested <= highest {
		return 0, fmt.Errorf("Requirement number %d is already used or reserved (high-water mark %d)", requested, highest)
	}
	if recovering {
		backup, err := restoreRequirementSequence(sequence, requested, errors.Is(sequenceErr, errSequenceCorrupt))
		if err != nil { return 0, err }
		fmt.Fprintf(os.Stderr, "Recovered Requirement sequence (%v): existing maximum=%d, allocated=%d, next=%s, backup=%s\n", sequenceErr, directoryHighest, requested, nextRequirementNumber(requested), backup)
		return requested, nil
	}
	file, err := os.OpenFile(sequence, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil { return 0, err }
	_, writeErr := fmt.Fprintf(file, "%d\n", requested)
	if writeErr == nil { writeErr = file.Sync() }
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil { return 0, fmt.Errorf("persist Requirement number: %w", err) }
	return requested, nil
}

// On Windows, enumerating a regular file can report ErrNotExist. Check the
// directory and its parents before treating a missing path as an empty source.
func readRequirementNumberingDirectory(path string) ([]os.DirEntry, error) {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() { return nil, fmt.Errorf("Requirement scan path is not a directory: %s", current) }
			if current != path { return nil, nil }
			// Once the directory exists, any enumeration failure must stop
			// allocation, including disappearance during the scan.
			return os.ReadDir(path)
		}
		if !errors.Is(err, os.ErrNotExist) { return nil, err }
		if filepath.Dir(current) == current { return nil, err }
	}
}

func nextRequirementNumber(number uint64) string {
	if number == ^uint64(0) { return "exhausted" }
	return strconv.FormatUint(number+1, 10)
}

// The caller owns the creation lock. Preserve corrupt bytes before replacement.
// Never remove the destination as a workaround for a failed rename.
func restoreRequirementSequence(path string, reserved uint64, damaged bool) (string, error) {
	backup := "none"
	if damaged {
		original, err := os.Open(path)
		if err != nil { return "", err }
		defer original.Close()
		backupDir := filepath.Join(filepath.Dir(path), "backups")
		if err := os.MkdirAll(backupDir, 0o700); err != nil { return "", err }
		file, err := os.CreateTemp(backupDir, "sequence-corrupt-*.bak")
		if err != nil { return "", err }
		backup = file.Name()
		_, copyErr := io.Copy(file, original)
		if copyErr == nil { copyErr = file.Sync() }
		if err := errors.Join(copyErr, file.Close()); err != nil { return backup, fmt.Errorf("back up Requirement sequence to %s: %w", backup, err) }
		if err := original.Close(); err != nil { return backup, err }
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".sequence-recovery-*.tmp")
	if err != nil { return backup, err }
	name := temporary.Name()
	defer os.Remove(name)
	_, writeErr := fmt.Fprintf(temporary, "%d\n", reserved)
	if writeErr == nil { writeErr = temporary.Sync() }
	if err := errors.Join(writeErr, temporary.Close()); err != nil { return backup, err }
	if err := os.Rename(name, path); err != nil { return backup, fmt.Errorf("replace Requirement sequence (backup=%s): %w", backup, err) }
	return backup, nil
}
