package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aiw/internal/ai"
)

// BackendFor is retained as a compatibility seam for Session callers while
// provider implementations live in the shared ai module.
func BackendFor(name, model string) (Backend, error) {
	cfg, err := ai.ConfigFor(name, model)
	if err != nil {
		return nil, err
	}
	return ai.NewProvider(cfg)
}

func SaveTurnResult(store *Store, status Status, result TurnResult) error {
	dir := filepath.Join(store.sessionDir(status.Session.ID), "outputs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	turn := status.Session.LastTurn + 1
	if err := atomicWrite(filepath.Join(dir, fmt.Sprintf("%04d-final.txt", turn)), []byte(result.FinalOutput)); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(dir, fmt.Sprintf("%04d-events.jsonl", turn)), result.Events); err != nil {
		return err
	}
	if len(result.Stderr) > 0 {
		if err := atomicWrite(filepath.Join(dir, fmt.Sprintf("%04d-stderr.log", turn)), result.Stderr); err != nil {
			return err
		}
	}
	if result.Usage != nil {
		ai.PrepareUsageEvidence(result.Usage, result.CompletedAt)
		usage, err := json.Marshal(result.Usage)
		if err != nil {
			return fmt.Errorf("encode turn usage: %w", err)
		}
		if err := atomicWrite(filepath.Join(dir, fmt.Sprintf("%04d-usage.json", turn)), usage); err != nil {
			return err
		}
	}
	return nil
}
