package execution

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"aiw/internal/workflow"
)

// DetachedAuxiliaryLauncher is an OS process launcher for an explicitly
// configured managed helper. Arguments are passed directly, without a shell;
// the helper receives the canonical project Store and Task ID. The helper must
// re-register the same controlled adapters and call AuxiliaryHost.Run. The
// executable is configuration owned by the host, never generated source text.
func DetachedAuxiliaryLauncher(executable string, arguments []string) (AuxiliaryLauncher, error) {
	if !filepath.IsAbs(executable) { return nil, errors.New("managed auxiliary helper must have an absolute executable path") }
	info, err := os.Stat(executable)
	if err != nil { return nil, err }
	if !info.Mode().IsRegular() { return nil, errors.New("managed auxiliary helper is not a regular executable") }
	fixed := append([]string(nil), arguments...)
	return func(store *workflow.Store, id workflow.TaskID) error {
		state, err := store.Load(id)
		if err != nil { return err }
		if state.Protocol == nil { return errors.New("auxiliary helper requires durable Task state") }
		root, err := filepath.Abs(store.Root)
		if err != nil { return err }
		args := append(append([]string(nil), fixed...), "--auxiliary-root", root, "--auxiliary-task", string(id))
		command := exec.Command(executable, args...)
		command.Dir = filepath.Dir(root)
		if err := detachAuxiliaryProcess(command); err != nil { return err }
		// Nil standard streams use the null device. No unbounded log file or
		// foreground pipe ties the helper to the launching Session.
		if err := command.Start(); err != nil { return err }
		return command.Process.Release()
	}, nil
}

// ConnectAuxiliary installs E05 on the same Store used by managed execution.
// It does not migrate a Task, create a policy, infer capability, or initialize
// an unknown historical resource balance. Those are explicit activation inputs.
func ConnectAuxiliary(store *workflow.Store, services workflow.AuxiliaryServices, launch AuxiliaryLauncher) error {
	if store == nil || launch == nil || services.FreeBytes == nil || services.Authorize == nil || services.VerifyCapability == nil || services.VerifyObservation == nil || services.VerifyInventory == nil || services.ReportGap == nil { return errors.New("complete auxiliary host, resource, authorization, and observation adapters are required") }
	services.StartHost = func(id workflow.TaskID) error { return launch(store, id) }
	store.AuxiliaryServices = &services
	return nil
}
