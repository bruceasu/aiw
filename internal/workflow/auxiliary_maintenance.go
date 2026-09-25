package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LocalAuxiliaryInventoryProof is an explicit maintenance attestation, paired
// with a fresh bounded filesystem snapshot. Missing ledgers alone NEVER prove
// no prior dispatch. Operators must explain the history, and an existing ledger
// is never overwritten by initialization.
type LocalAuxiliaryInventoryProof struct {
	Version int `json:"version"`
	Reason string `json:"reason"`
	ReviewedBy string `json:"reviewed_by"`
	NoPriorDispatches bool `json:"no_prior_dispatches"`
	PriorDispatches bool `json:"prior_dispatches"`
	TreeDigest string `json:"tree_digest"`
	TaskBytes map[TaskID]int64 `json:"task_bytes"`
	ProjectBytes int64 `json:"project_bytes"`
}

// InspectLocalAuxiliaryInventory is a read-only, bounded maintenance scan. All
// pre-existing runtime bytes are conservatively included, not just outputs.
// The snapshot excludes only the two installation documents and OS lock files.
// Symlinks/reparse targets and unreadable history fail closed.
func (s *Store) InspectLocalAuxiliaryInventory() (LocalAuxiliaryInventoryProof, error) {
	proof := LocalAuxiliaryInventoryProof{Version: 1, TaskBytes: map[TaskID]int64{}}
	root, err := filepath.Abs(s.Root)
	if err != nil { return proof, err }
	root, err = filepath.EvalSymlinks(root)
	if err != nil { return proof, err }
	deadline := time.Now().Add(120*time.Second)
	var files []string
	count := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil { return visitErr }
		count++
		if count > 16384 || time.Now().After(deadline) { return errors.New("auxiliary inventory scan bound exceeded") }
		if entry.Type()&os.ModeSymlink != 0 { return errors.New("auxiliary inventory requires local non-linked files") }
		rel, err := filepath.Rel(root, path)
		if err != nil { return err }
		rel = filepath.ToSlash(rel)
		if rel == "locks" && entry.IsDir() { return filepath.SkipDir }
		if entry.IsDir() { return nil }
		if rel == "auxiliary-inventory.json" || rel == "resource-policy.json" { return nil }
		info, err := entry.Info()
		if err != nil { return err }
		if !info.Mode().IsRegular() { return errors.New("unsupported auxiliary inventory file type") }
		files = append(files, rel)
		if entry.Name() != runtimeStateFile { return nil }
		parts := strings.Split(rel, "/")
		if len(parts) != 2 && !(len(parts) == 3 && parts[0] == runtimeTasksDir) { return nil }
		id := TaskID(parts[len(parts)-2])
		if validateTaskID(id) != nil { return errors.New("invalid Task in auxiliary inventory") }
		if _, duplicate := proof.TaskBytes[id]; duplicate { return errors.New("duplicate legacy/canonical Task inventory requires reconciliation") }
		if info.Size() > 16*1024*1024 { return errors.New("Task state exceeds inventory read bound") }
		data, err := os.ReadFile(path)
		if err != nil { return err }
		var state RuntimeState
		if err := json.Unmarshal(data, &state); err != nil { return err }
		if state.Task.ID != id || state.PendingEvent != nil || state.WriteLease != nil || state.Automation.PreparedRequest != nil || state.Automation.Supervisor.LeaseID != "" || activeAttempt(state.Attempts) != "" { return errors.New("auxiliary inventory requires reconciled idle Tasks") }
		if state.Protocol != nil && state.Protocol.Auxiliary != nil {
			for _, job := range state.Protocol.Auxiliary.Jobs {
				if len(job.Calls) != 0 || job.Result != nil || job.Output != nil || job.RecoveryUsed || job.PublishAttempts != 0 { proof.PriorDispatches = true }
			}
		}
		proof.TaskBytes[id] = 0
		return nil
	})
	if err != nil { return proof, err }
	if len(proof.TaskBytes) == 0 { return proof, errors.New("auxiliary inventory contains no managed Tasks") }
	sort.Strings(files)
	hash := sha256.New()
	physical := map[int64][]os.FileInfo{}
	for _, rel := range files {
		if time.Now().After(deadline) { return proof, errors.New("auxiliary inventory deadline exceeded") }
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil { return proof, err }
		if !info.Mode().IsRegular() { return proof, errors.New("inventory changed file type") }
		fmt.Fprintf(hash, "%s\x00%d\x00%d\n", rel, info.Size(), info.ModTime().UnixNano())
		if filepath.Base(rel) == runtimeStateFile {
			if info.Size() > 16*1024*1024 { return proof, errors.New("Task state exceeds inventory bound") }
			data, err := os.ReadFile(path)
			if err != nil { return proof, err }
			fmt.Fprintln(hash, contentDigest(data))
		}
		shared := false
		for _, old := range physical[info.Size()] { if os.SameFile(old, info) { shared = true; break } }
		if shared { continue }
		physical[info.Size()] = append(physical[info.Size()], info)
		proof.ProjectBytes += info.Size()
		if proof.ProjectBytes > 4*1024*1024*1024 { return proof, errors.New("existing runtime inventory exceeds the R3 project ceiling; preserve it") }
		parts := strings.Split(rel, "/")
		id := TaskID(parts[0])
		if parts[0] == runtimeTasksDir && len(parts) > 1 { id = TaskID(parts[1]) }
		if _, known := proof.TaskBytes[id]; known { proof.TaskBytes[id] += info.Size() }
	}
	proof.TreeDigest = hex.EncodeToString(hash.Sum(nil))
	return proof, nil
}

func (s *Store) VerifyLocalAuxiliaryInventory(inventory AuxiliaryInventory) error {
	if inventory.Evidence.Kind != "auxiliary-inventory" || inventory.Evidence.Path != "auxiliary-inventory.json" { return errors.New("inventory needs the managed local evidence document") }
	data, err := os.ReadFile(filepath.Join(s.Root, inventory.Evidence.Path))
	if err != nil { return err }
	if len(data) > 64*1024 || contentDigest(data) != inventory.Evidence.SHA256 { return errors.New("inventory evidence version is unavailable") }
	var proof LocalAuxiliaryInventoryProof
	if err := strictJSON(data, &proof); err != nil { return err }
	if proof.Version != 1 || proof.Reason == "" || proof.ReviewedBy == "" || !proof.NoPriorDispatches { return errors.New("initialization requires explicit reviewed no-prior-dispatch history") }
	current, err := s.InspectLocalAuxiliaryInventory()
	if err != nil { return err }
	if current.PriorDispatches || proof.PriorDispatches { return errors.New("prior auxiliary execution cannot be initialized as zero usage") }
	a, _ := json.Marshal(current.TaskBytes)
	b, _ := json.Marshal(proof.TaskBytes)
	c, _ := json.Marshal(inventory.TaskBytes)
	if current.TreeDigest != proof.TreeDigest || current.ProjectBytes != proof.ProjectBytes || inventory.ProjectBytes != proof.ProjectBytes || string(a) != string(b) || string(a) != string(c) { return errors.New("auxiliary inventory changed; obtain a fresh reviewed snapshot") }
	return nil
}

// InitializeLocalAuxiliaryResources never enables schema 10 or the Worker. It
// consumes an operator-reviewed snapshot already installed at the fixed path.
func (s *Store) InitializeLocalAuxiliaryResources() error {
	data, err := os.ReadFile(filepath.Join(s.Root, "auxiliary-inventory.json"))
	if err != nil { return err }
	if len(data) > 64*1024 { return errors.New("inventory document exceeds bound") }
	var proof LocalAuxiliaryInventoryProof
	if err := strictJSON(data, &proof); err != nil { return err }
	if _, _, err := s.auxiliaryPolicy(); err != nil { return err }
	return s.InitializeAuxiliaryResources(AuxiliaryInventory{Known: true, Evidence: ActorReference{Kind: "auxiliary-inventory", Path: "auxiliary-inventory.json", SHA256: contentDigest(data)}, TaskBytes: proof.TaskBytes, ProjectBytes: proof.ProjectBytes})
}

func (s *Store) InstallAuxiliaryPolicy(policy AuxiliaryResourcePolicy) error {
	lock, err := s.auxiliaryProjectLock()
	if err != nil { return err }
	defer systemTaskUnlock(lock)
	if err := validateAuxiliaryPolicy(policy); err != nil { return err }
	old, _, err := s.auxiliaryPolicy()
	if err != nil && !errors.Is(err, os.ErrNotExist) { return err }
	if err == nil && policy.Version <= old.Version { return errors.New("policy update requires a newer version and reason") }
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil { return err }
	return durableWrite(filepath.Join(s.Root, "resource-policy.json"), append(data, '\n'))
}

// SettleLocalAuxiliaryStorage refreshes physical-byte baselines and releases
// only terminal queue storage peaks. Model usage, request identity, source
// recovery, policy history and every unknown reservation are retained.
func (s *Store) SettleLocalAuxiliaryStorage() error {
	lock, err := s.auxiliaryProjectLock()
	if err != nil { return err }
	defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources()
	if err != nil { return err }
	if r.Active != "" { return errors.New("reconcile the active auxiliary call before storage settlement") }
	data, err := os.ReadFile(filepath.Join(s.Root, "auxiliary-inventory.json"))
	if err != nil { return err }
	if len(data) > 64*1024 { return errors.New("inventory evidence exceeds bound") }
	var proof LocalAuxiliaryInventoryProof
	if err := strictJSON(data, &proof); err != nil { return err }
	if proof.Version != 1 || proof.ReviewedBy == "" || proof.Reason == "" { return errors.New("storage settlement requires a reviewed inventory") }
	current, err := s.InspectLocalAuxiliaryInventory()
	if err != nil { return err }
	a, _ := json.Marshal(current.TaskBytes)
	b, _ := json.Marshal(proof.TaskBytes)
	if current.TreeDigest != proof.TreeDigest || current.ProjectBytes != proof.ProjectBytes || current.PriorDispatches != proof.PriorDispatches || string(a) != string(b) { return errors.New("storage inventory changed") }
	for id := range r.Inventory.TaskBytes { if _, ok := current.TaskBytes[id]; !ok { return errors.New("storage inventory cannot forget an existing source Task") } }
	for key, q := range r.Queue {
		// Source seals have their own immutable artifact, not a model job.
		// Keep their conservative peak until that dedicated lifecycle settles.
		if q.Kind == "source-seal" { continue }
		state, err := s.Load(q.Owner)
		if err != nil { return err }
		job, err := findAuxiliaryJob(state, key)
		if err != nil { return err }
		if q.Terminal && auxiliaryTerminal(job.State) { q.PeakBytes = 0; r.Queue[key] = q }
	}
	r.Inventory = AuxiliaryInventory{Known: true, Evidence: ActorReference{Kind: "auxiliary-inventory", Path: "auxiliary-inventory.json", SHA256: contentDigest(data)}, TaskBytes: current.TaskBytes, ProjectBytes: current.ProjectBytes}
	r.Inventory, err = s.freezeLocalAuxiliaryInventory(r.Inventory)
	if err != nil { return err }
	return s.saveAuxiliaryResources(r)
}

func (s *Store) freezeLocalAuxiliaryInventory(inventory AuxiliaryInventory) (AuxiliaryInventory, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, "auxiliary-inventory.json"))
	if err != nil { return inventory, err }
	if len(data) > 64*1024 || contentDigest(data) != inventory.Evidence.SHA256 { return inventory, errors.New("inventory evidence changed before sealing") }
	relative := "auxiliary-inventories/"+inventory.Evidence.SHA256+".json"
	path := filepath.Join(s.Root, filepath.FromSlash(relative))
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// Both the immutable copy and temporary replacement fit the already
		// reserved project control headroom. Future scans count the copy.
		if s.AuxiliaryServices == nil || s.AuxiliaryServices.FreeBytes == nil { return inventory, errors.New("inventory volume capacity unavailable") }
		free, err := s.AuxiliaryServices.FreeBytes(s.Root)
		if err != nil { return inventory, err }
		if free-int64(2*len(data))-auxiliaryControlBytes < 256*1024*1024 || inventory.ProjectBytes+int64(len(data))+auxiliaryControlBytes > 4*1024*1024*1024 { return inventory, errors.New("inventory seal exceeds storage allowance") }
		if err := durableWrite(path, data); err != nil { return inventory, err }
		inventory.ProjectBytes += int64(len(data))
	} else if err != nil { return inventory, err } else {
		prior, err := os.ReadFile(path)
		if err != nil { return inventory, err }
		if contentDigest(prior) != inventory.Evidence.SHA256 { return inventory, errors.New("immutable inventory evidence conflict") }
	}
	inventory.Evidence.Path = relative
	return inventory, nil
}
