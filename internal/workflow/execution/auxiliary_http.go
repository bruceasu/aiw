package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"aiw/internal/workflow"
)

// HTTPAuxiliaryWorker has no shell, tools, filesystem action language or child
// process API. Its one side effect is one bounded HTTP POST. There is no SDK
// retry, redirect, proxy inheritance or asynchronous provider job. Loss of the
// response is UNKNOWN, even after the local request context has been cancelled.
type HTTPAuxiliaryWorker struct {
	Store *workflow.Store
	ID string
}

func (w *HTTPAuxiliaryWorker) Identity() string { return w.ID }

type auxiliaryHTTPJournal struct {
	Version int `json:"version"`
	Adapter string `json:"adapter"`
	Owner workflow.TaskID `json:"owner"`
	Measurement workflow.AuxiliaryMeasurement `json:"measurement"`
	Observation workflow.AuxiliaryObservation `json:"observation"`
}

func auxiliaryHTTPBody(job workflow.AuxiliaryJob, record auxiliaryCapabilityRecord, output int64) ([]byte, error) {
	// This is the entire provider request. The evidence-backed counter also
	// includes the provider's framing bound; nothing adds a hidden system prompt.
	return json.Marshal(struct {
		Model string `json:"model"`
		Messages []map[string]string `json:"messages"`
		Stream bool `json:"stream"`
		MaxTokens int64 `json:"max_tokens"`
		N int `json:"n"`
	}{record.Selection.Model, []map[string]string{{"role": "user", "content": job.Prompt}}, false, output, 1})
}

func (w *HTTPAuxiliaryWorker) Measure(job workflow.AuxiliaryJob) (workflow.AuxiliaryMeasurement, error) {
	var measurement workflow.AuxiliaryMeasurement
	config, record, identity, err := loadAuxiliaryConfig(w.Store)
	if err != nil { return measurement, err }
	if identity != w.ID { return measurement, errors.New("auxiliary configuration changed before measurement") }
	if os.Getenv(config.APIKeyEnv) == "" { return measurement, errors.New("auxiliary credential is unavailable") }
	output := min(int64(workflow.AuxiliaryOutputTokens), record.OutputTokens)
	body, err := auxiliaryHTTPBody(job, record, output)
	if err != nil { return measurement, err }
	input := int64(len(body))+record.EnvelopeTokens
	if !utf8.ValidString(job.Prompt) || auxiliaryDigest([]byte(job.Prompt)) != job.InputDigest || len(body) > workflow.AuxiliaryInputBytes || input > workflow.AuxiliaryInputTokens || input+output > record.ContextTokens { return measurement, errors.New("complete auxiliary provider input exceeds the verified capability or R3 allowance") }
	return workflow.AuxiliaryMeasurement{Capability: capabilityFromRecord(config, record), PromptDigest: job.InputDigest, InputTokens: input, OutputTokens: output}, nil
}

func (w *HTTPAuxiliaryWorker) journal(call workflow.AuxiliaryReservation, observation workflow.AuxiliaryObservation) (*workflow.AuxiliaryObservation, error) {
	observation.RequestID, observation.Key = call.ID, call.Key
	observation.Executor, observation.InputDigest = call.Executor, call.Measurement.PromptDigest
	observation.Evidence = workflow.ActorReference{}
	entry := auxiliaryHTTPJournal{1, "http-closed-input-v1", call.Owner, call.Measurement, observation}
	ref, err := w.Store.PersistAuxiliaryEvidence(call, entry)
	if err != nil { return nil, err }
	observation.Evidence = ref
	return &observation, nil
}

func verifyAuxiliaryHTTPObservation(store *workflow.Store, call workflow.AuxiliaryReservation, observation workflow.AuxiliaryObservation) error {
	if !strings.HasPrefix(call.Executor, auxiliaryAdapter) || observation.Executor != call.Executor { return errors.New("unsupported auxiliary observation executor") }
	var entry auxiliaryHTTPJournal
	if err := store.ReadExecutionArtifact(call.Owner, observation.Evidence, &entry); err != nil { return err }
	if entry.Version != 1 || entry.Adapter != "http-closed-input-v1" || entry.Owner != call.Owner || entry.Measurement != call.Measurement { return errors.New("auxiliary journal identity or capability changed") }
	expected := observation
	expected.Evidence = workflow.ActorReference{}
	if entry.Observation != expected || expected.RequestID != call.ID || expected.Key != call.Key || expected.InputDigest != call.Measurement.PromptDigest { return errors.New("auxiliary observation differs from the exact host journal") }
	return nil
}

func (w *HTTPAuxiliaryWorker) Reconcile(_ context.Context, call workflow.AuxiliaryReservation) (*workflow.AuxiliaryObservation, error) {
	if call.Executor != w.ID { return nil, errors.New("auxiliary reconciliation executor mismatch") }
	// Read the reservation again: a crash may have occurred after persisting
	// terminal evidence but before committing the Task observation.
	active, err := w.Store.AuxiliaryActiveCall()
	if err != nil { return nil, err }
	if active == nil || active.ID != call.ID { return nil, errors.New("original auxiliary reservation is no longer active") }
	for i := len(active.Evidence)-1; i >= 0; i-- {
		var entry auxiliaryHTTPJournal
		if err := w.Store.ReadExecutionArtifact(call.Owner, active.Evidence[i], &entry); err != nil {
			// A reserved but unwritten terminal artifact is not proof of a
			// completed request. Older evidence can only keep it unknown.
			if errors.Is(err, os.ErrNotExist) { continue }
			return nil, err
		}
		observation := entry.Observation
		observation.Evidence = active.Evidence[i]
		if err := verifyAuxiliaryHTTPObservation(w.Store, call, observation); err != nil { return nil, err }
		return &observation, nil
	}
	return nil, errors.New("original auxiliary request has no reliable dispatch observation; retain its slot and full reservation")
}

func (w *HTTPAuxiliaryWorker) Start(parent context.Context, job workflow.AuxiliaryJob, call workflow.AuxiliaryReservation) (*workflow.AuxiliaryObservation, error) {
	active, err := w.Store.AuxiliaryActiveCall()
	if err != nil { return nil, err }
	if active == nil || active.ID != call.ID || active.Executor != w.ID { return nil, errors.New("auxiliary dispatch is not the active fixed request") }
	if len(active.Evidence) != 0 { return w.Reconcile(parent, call) }
	notSent := func(reason string) (*workflow.AuxiliaryObservation, error) {
		return w.journal(call, workflow.AuxiliaryObservation{State: "not-dispatched", Reason: reason})
	}
	measurement, err := w.Measure(job)
	if err != nil { return notSent(err.Error()) }
	if measurement != call.Measurement { return notSent("fixed auxiliary measurement changed") }
	config, record, identity, err := loadAuxiliaryConfig(w.Store)
	if err != nil || identity != w.ID { return notSent("auxiliary configuration changed") }
	key := os.Getenv(config.APIKeyEnv)
	if key == "" { return notSent("auxiliary credential is unavailable") }
	body, err := auxiliaryHTTPBody(job, record, measurement.OutputTokens)
	if err != nil { return notSent("auxiliary request encoding failed") }
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Endpoint, bytes.NewReader(body))
	if err != nil { return notSent("auxiliary request construction failed") }
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	if _, err := w.Store.CheckAuxiliaryDispatch(call); err != nil { return notSent("auxiliary authorization or Stop changed before dispatch") }
	if ctx.Err() != nil { return notSent("auxiliary host deadline reached before dispatch") }
	// Persist BEFORE sending. A crash in this interval stays unknown, not a
	// guess that nothing happened. Host fencing prevents another Start racing it.
	if _, err := w.journal(call, workflow.AuxiliaryObservation{State: "unknown", Reason: "HTTP dispatch intent; no terminal response observed"}); err != nil { return nil, err }
	if _, err := w.Store.CheckAuxiliaryDispatch(call); err != nil { return notSent("Stop or authorization changed after intent and before HTTP send") }
	transport := &http.Transport{
		Proxy: nil, DisableKeepAlives: true, DisableCompression: true,
		DialContext: (&net.Dialer{Timeout: 15*time.Second}).DialContext,
		TLSHandshakeTimeout: 15*time.Second, ResponseHeaderTimeout: 120*time.Second,
		MaxResponseHeaderBytes: 16*1024,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 120*time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil { return w.journal(call, workflow.AuxiliaryObservation{State: "unknown", Reason: "HTTP response unavailable; remote termination is unproven"}) }
	defer response.Body.Close()
	// Allow JSON escaping overhead, but never unbounded provider output or
	// logs. Oversize/truncated/error replies do not prove remote termination.
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8*workflow.AuxiliaryOutputBytes+1))
	if err != nil || len(raw) > 8*workflow.AuxiliaryOutputBytes || response.StatusCode != http.StatusOK {
		return w.journal(call, workflow.AuxiliaryObservation{State: "unknown", Reason: "incomplete, oversized or non-success HTTP response; retain reservation"})
	}
	var decoded struct {
		Model string `json:"model"`
		Choices []struct {
			Finish string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
				Tools json.RawMessage `json:"tool_calls"`
				Function json.RawMessage `json:"function_call"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			Input *int64 `json:"prompt_tokens"`
			Output *int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &decoded) != nil || len(decoded.Choices) != 1 || decoded.Choices[0].Finish == "" {
		return w.journal(call, workflow.AuxiliaryObservation{State: "unknown", Reason: "no reliable terminal generation response"})
	}
	choice := decoded.Choices[0]
	switch choice.Finish {
	case "stop", "length", "content_filter", "tool_calls", "function_call":
	default:
		return w.journal(call, workflow.AuxiliaryObservation{State: "unknown", Reason: "unrecognized provider termination semantics"})
	}
	if decoded.Model != record.Selection.Model {
		return w.journal(call, workflow.AuxiliaryObservation{State: "unknown", Reason: "provider returned an unreviewed model identity; reconcile capability"})
	}
	observation := workflow.AuxiliaryObservation{State: "invalid", Reason: "terminal response is not a complete bounded text result"}
	if choice.Finish == "stop" && len(choice.Message.Tools) == 0 && len(choice.Message.Function) == 0 && strings.TrimSpace(choice.Message.Content) != "" && utf8.ValidString(choice.Message.Content) && len(choice.Message.Content) <= workflow.AuxiliaryOutputBytes {
		observation.State, observation.Text, observation.Reason = "valid", choice.Message.Content, ""
		if err := workflow.ValidateKnowledgeOutput(job, observation.Text); err != nil { observation.State, observation.Text, observation.Reason = "invalid", "", err.Error() }
		if observation.State == "valid" {
			if err := workflow.ValidateVerifierOutput(job, observation.Text); err != nil { observation.State, observation.Text, observation.Reason = "invalid", "", err.Error() }
		}
	}
	if usage := decoded.Usage; usage != nil && usage.Input != nil && usage.Output != nil {
		if *usage.Input >= 0 && *usage.Output >= 0 && *usage.Input <= measurement.InputTokens && *usage.Output <= measurement.OutputTokens {
			observation.UsageKnown, observation.InputTokens, observation.OutputTokens = true, *usage.Input, *usage.Output
		} else {
			// A violated capability does not authorize another generation with
			// the same broken bound. Unknown keeps the project slot quarantined.
			observation = workflow.AuxiliaryObservation{State: "unknown", Reason: "provider usage exceeds its verified reservation; capability reconciliation required"}
		}
	}
	return w.journal(call, observation)
}
