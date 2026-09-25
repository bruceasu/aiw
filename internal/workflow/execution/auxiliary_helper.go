package execution

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aiw/internal/notification"
	"aiw/internal/workflow"
)

func ProjectSessionTaskMemory(id string) (string, error) {
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(id))
	if err != nil { return "", err }
	if state.Protocol == nil { return "", nil }
	store.ResumeAuxiliaryHost(workflow.TaskID(id))
	text, _, err := store.TaskMemoryContext(workflow.TaskID(id))
	return text, err
}

// RunAuxiliaryHelper is the private child-process entry. Only the fixed root
// and Task identity cross the process boundary; no prompt, shell or credential
// is placed on its command line. A single host lock also fences concurrent
// foreground launchers for different Tasks in this project.
func RunAuxiliaryHelper(args []string) error {
	flags := flag.NewFlagSet("auxiliary-helper", flag.ContinueOnError)
	root := flags.String("auxiliary-root", "", "canonical local project runtime directory")
	task := flags.String("auxiliary-task", "", "source Task ID")
	if err := flags.Parse(args); err != nil { return err }
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || *task == "" || *task == "." || *task == ".." || strings.ContainsAny(*task, `/\\`) { return errors.New("auxiliary helper requires an absolute root and one Task ID") }
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil { return err }
	if strings.HasPrefix(canonical, `\\`) { return errors.New("auxiliary helper requires the verified local-disk boundary") }
	store := workflow.NewStore(canonical)
	ConfigureProductionAuxiliary(store)
	// A child never launches another helper. It owns the bounded project drain.
	store.AuxiliaryServices.StartHost = func(workflow.TaskID) error { return nil }
	id := workflow.TaskID(*task)
	state, err := store.Load(id)
	if err != nil { return err }
	if state.Protocol == nil { return errors.New("auxiliary helper cannot activate a legacy Task") }
	release, err := store.LockAuxiliaryHost()
	if err != nil { return err }
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	workers := map[string]AuxiliaryWorker{}
	config, _, identity, configErr := loadAuxiliaryConfig(store)
	if configErr == nil {
		worker := &HTTPAuxiliaryWorker{Store: store, ID: identity}
		workers["memory"] = worker
		workers["knowledge-extraction"] = worker
		workers["knowledge-summary"] = worker
		workers["verifier"] = worker
	}
	active, err := store.AuxiliaryActiveCall()
	if err != nil { configErr = errors.Join(configErr, err) }
	if active != nil && strings.HasPrefix(active.Executor, auxiliaryAdapter) {
		// Read the original journal even if config is disabled or replaced.
		workers["original-observer"] = &HTTPAuxiliaryWorker{Store: store, ID: active.Executor}
	}
	ids := []workflow.TaskID{id}
	for other := range config.Tasks { if other != id { ids = append(ids, other) } }
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var failures []error
	for _, taskID := range ids {
		if ctx.Err() != nil { break }
		state, err := store.Load(taskID)
		if err != nil || state.Protocol == nil { continue }
		// The notification lane is independent of model capability and uses
		// this helper's project lock. It may finish after model work does.
		notices := make(chan error, 1)
		go func(taskID workflow.TaskID) { notices <- notification.Run(ctx, store, taskID) }(taskID)
		runErr := (AuxiliaryHost{Store: store, Workers: workers}).Run(ctx, taskID)
		runErr = errors.Join(runErr, <-notices)
		gap := errors.Join(configErr, runErr)
		if err := store.RecordAuxiliaryHostGap(taskID, gap); err != nil { failures = append(failures, err) }
		if gap != nil { failures = append(failures, gap) }
	}
	return errors.Join(append(failures, ctx.Err())...)
}

// RunAuxiliaryMaintenance is called by the managed workflow command parser.
// It does not launch helpers, mutate Task lifecycle or create model capability.
func RunAuxiliaryMaintenance(args []string) error {
	if len(args) == 0 { return errors.New("usage: workflow auxiliary <inventory|initialize|settle|policy> [policy-file]") }
	store := workflow.NewStore("")
	ConfigureProductionAuxiliary(store)
	store.AuxiliaryServices.StartHost = func(workflow.TaskID) error { return nil }
	switch args[0] {
	case "inventory":
		if len(args) != 1 { return errors.New("usage: workflow auxiliary inventory") }
		proof, err := store.InspectLocalAuxiliaryInventory()
		if err != nil { return err }
		return json.NewEncoder(os.Stdout).Encode(proof)
	case "initialize":
		if len(args) != 1 { return errors.New("usage: workflow auxiliary initialize") }
		release, err := store.LockAuxiliaryHost()
		if err != nil { return err }
		defer release()
		return store.InitializeLocalAuxiliaryResources()
	case "settle":
		if len(args) != 1 { return errors.New("usage: workflow auxiliary settle") }
		release, err := store.LockAuxiliaryHost()
		if err != nil { return err }
		defer release()
		return store.SettleLocalAuxiliaryStorage()
	case "policy":
		if len(args) != 2 { return errors.New("usage: workflow auxiliary policy <reviewed-policy.json>") }
		var policy workflow.AuxiliaryResourcePolicy
		if _, err := readAuxiliaryJSON(args[1], &policy); err != nil { return err }
		return store.InstallAuxiliaryPolicy(policy)
	default:
		return fmt.Errorf("unknown auxiliary maintenance operation %q", args[0])
	}
}
