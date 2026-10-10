package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const maxRPCLine = 1 << 20

// Windows inherited-handle flags must only be changed during process start.
var processStartMu sync.Mutex

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcRequestError struct { submitted bool; cause error }
func (err *rpcRequestError) Error() string { return err.cause.Error() }
func (err *rpcRequestError) Unwrap() error { return err.cause }

func rpcSubmitted(err error) bool {
	var requestErr *rpcRequestError
	return errors.As(err, &requestErr) && requestErr.submitted
}

type rpcRead struct {
	message rpcMessage
	err     error
}

// appRPC confines JSON-RPC framing and ID routing to the backend process.
// Calls are serialized because each pool process owns one active request.
type appRPC struct {
	input  io.WriteCloser
	reads  chan rpcRead
	events chan rpcMessage
	write  sync.Mutex
	call   sync.Mutex
	nextID uint64
	closed chan struct{}
	close  sync.Once
}

type appProcess struct {
	child *process
	rpc   *appRPC
	stdout *os.File
	stderr *os.File
	stopOnce sync.Once
}

var appFeatureOverrides = []string{
	"features.shell_tool=false",
	"features.unified_exec=false",
	"features.code_mode=false",
	"features.code_mode_host=false",
	"features.code_mode_only=false",
	"features.view_image=false",
	"features.sleep_tool=false",
	"features.standalone_web_search=false",
	"features.web_search_request=false",
	"features.apps=false",
	"features.enable_mcp_apps=false",
	"features.plugins=false",
	"features.remote_plugin=false",
	"features.plugin_sharing=false",
	"features.tool_suggest=false",
	"features.recommended_plugins=false",
	"features.in_app_local_automation=false",
	"features.hooks=false",
	"features.multi_agent=false",
	"features.multi_agent_v2=false",
	"features.browser_use=false",
	"features.browser_use_full_cdp_access=false",
	"features.browser_use_external=false",
	"features.computer_use=false",
	"features.image_generation=false",
	"features.request_permissions_tool=false",
	"features.memory_tool=false",
	"features.artifact=false",
	"web_search=disabled",
}

func startAppProcess(c Config) (*appProcess, error) {
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil { return nil, errors.New("app-server stdout pipe failed") }
	stdinRead, stdinWrite, err := os.Pipe()
	if err != nil { stdoutRead.Close(); stdoutWrite.Close(); return nil, errors.New("app-server stdin pipe failed") }
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil { stdoutRead.Close(); stdoutWrite.Close(); stdinRead.Close(); stdinWrite.Close(); return nil, errors.New("app-server stderr pipe failed") }
	args := make([]string, 0, len(appFeatureOverrides)*2+3)
	for _, override := range appFeatureOverrides { args = append(args, "-c", override) }
	args = append(args, "app-server", "--stdio")
	if c.CodexScript != "" { args = append([]string{c.CodexScript}, args...) }
	processStartMu.Lock()
	child, err := startProcess(c.CodexPath, args, childEnv(), c.WorkspaceDir, stdinRead, stdoutWrite, stderrWrite)
	processStartMu.Unlock()
	stdinRead.Close(); stdoutWrite.Close(); stderrWrite.Close()
	if err != nil {
		stdoutRead.Close(); stdinWrite.Close(); stderrRead.Close()
		return nil, errors.New("app-server process start failed")
	}
	app := &appProcess{child: child, stdout: stdoutRead, stderr: stderrRead}
	app.rpc = newAppRPC(stdoutRead, stdinWrite)
	go func() {
		read, _ := io.Copy(io.Discard, io.LimitReader(stderrRead, maxWire+1))
		if read > maxWire { _ = child.stopTree() }
	}()
	return app, nil
}

func (app *appProcess) stop() error {
	if app == nil { return nil }
	var stopErr error
	app.stopOnce.Do(func() {
		if app.rpc != nil { app.rpc.stop() }
		if app.stdout != nil { _ = app.stdout.Close() }
		if app.stderr != nil { _ = app.stderr.Close() }
		if app.child != nil { stopErr = app.child.stopTree(); app.child.close() }
	})
	return stopErr
}

func (app *appProcess) preflight(ctx context.Context) error {
	result, err := app.rpc.request(ctx, "initialize", map[string]any{
		"clientInfo": map[string]string{"name": "aiw-agent-gateway", "version": "1"},
		"capabilities": map[string]any{"experimentalApi": true},
	})
	if err != nil { return err }
	var initialized struct {
		UserAgent string `json:"userAgent"`
		CodexHome string `json:"codexHome"`
		PlatformFamily string `json:"platformFamily"`
		PlatformOS string `json:"platformOs"`
	}
	if json.Unmarshal(result, &initialized) != nil || initialized.UserAgent == "" || initialized.CodexHome == "" || initialized.PlatformFamily == "" || initialized.PlatformOS == "" { return errors.New("app-server initialize capabilities are unknown") }
	if err := app.rpc.notify("initialized", map[string]any{}); err != nil { return err }
	return app.verifySafety(ctx)
}

func (app *appProcess) verifySafety(ctx context.Context) error {
	config, err := app.rpc.request(ctx, "config/read", map[string]any{"includeLayers": false})
	if err != nil || !json.Valid(config) || len(config) == 0 { return errors.New("app-server effective config is unavailable") }
	if unsafeConfiguredMCP(config) { return errors.New("app-server config contains enabled or unknown MCP servers") }
	status, err := app.rpc.request(ctx, "mcpServerStatus/list", map[string]any{})
	if err != nil || !safeMCPStatus(status) { return errors.New("app-server MCP status is enabled or unknown") }
	return nil
}

func unsafeConfiguredMCP(data []byte) bool {
	var value any
	if json.Unmarshal(data, &value) != nil { return true }
	var inspect func(any) bool
	inspect = func(current any) bool {
		switch object := current.(type) {
		case map[string]any:
			for key, child := range object {
				if key == "mcp_servers" || key == "mcpServers" {
					servers, ok := child.(map[string]any)
					if !ok { return true }
					for _, raw := range servers {
						config, ok := raw.(map[string]any)
						if !ok { return true }
						enabled, hasEnabled := config["enabled"].(bool)
						disabled, hasDisabled := config["disabled"].(bool)
						if !((hasEnabled && !enabled) || (hasDisabled && disabled)) { return true }
					}
				}
				if inspect(child) { return true }
			}
		case []any:
			for _, child := range object { if inspect(child) { return true } }
		}
		return false
	}
	if inspect(value) { return true }
	return false
}

func safeMCPStatus(data []byte) bool {
	var value any
	if json.Unmarshal(data, &value) != nil { return false }
	var servers []any
	switch result := value.(type) {
	case []any:
		servers = result
	case map[string]any:
		list, ok := result["data"].([]any)
		if !ok { list, ok = result["servers"].([]any) }
		if !ok { return false }
		servers = list
	default:
		return false
	}
	for _, raw := range servers {
		server, ok := raw.(map[string]any)
		if !ok { return false }
		state, _ := server["status"].(string)
		enabled, hasEnabled := server["enabled"].(bool)
		if state == "disabled" { continue }
		if state != "" || !hasEnabled || enabled { return false }
	}
	return true
}

func newAppRPC(stdout io.Reader, input io.WriteCloser) *appRPC {
	rpc := &appRPC{input: input, reads: make(chan rpcRead, 64), events: make(chan rpcMessage, 32), closed: make(chan struct{})}
	go rpc.readLoop(stdout)
	return rpc
}

func (rpc *appRPC) readLoop(stdout io.Reader) {
	reader := bufio.NewReaderSize(stdout, 32<<10)
	buffer := make([]byte, 0, 32<<10)
	total := 0
	for {
		part, err := reader.ReadSlice('\n')
		total += len(part)
		if total > maxWire {
			rpc.deliver(rpcRead{err: errors.New("app-server output exceeds process limit")})
			return
		}
		if len(buffer)+len(part) > maxRPCLine {
			rpc.deliver(rpcRead{err: errors.New("app-server JSONL line exceeds limit")})
			return
		}
		buffer = append(buffer, part...)
		if err == nil {
			line := buffer[:len(buffer)-1]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			var message rpcMessage
			if json.Unmarshal(line, &message) != nil || message.JSONRPC != "2.0" {
				rpc.deliver(rpcRead{err: errors.New("invalid app-server JSON-RPC message")})
				return
			}
			if message.Method == "" && len(message.ID) == 0 {
				rpc.deliver(rpcRead{err: errors.New("app-server response has no ID")})
				return
			}
			if message.Method != "" && len(message.ID) == 0 {
				select {
				case rpc.events <- message:
				default:
					rpc.deliver(rpcRead{err: errors.New("app-server notification buffer exceeded")})
					return
				}
				buffer = buffer[:0]
				continue
			}
			if message.Method != "" && len(message.ID) > 0 {
				// The server is asking the gateway to execute a capability. Reject
				// it explicitly and fail the process; no server tool is exposed.
				_ = rpc.writeMessage(map[string]any{"jsonrpc": "2.0", "id": message.ID, "error": map[string]any{"code": -32601, "message": "server requests are unsupported"}})
				rpc.deliver(rpcRead{err: fmt.Errorf("unexpected app-server request: %s", message.Method)})
				return
			}
			rpc.deliver(rpcRead{message: message})
			buffer = buffer[:0]
			continue
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			_ = err
			rpc.deliver(rpcRead{err: errors.New("app-server JSONL stream ended unexpectedly")})
			return
		}
	}
}

func (rpc *appRPC) deliver(read rpcRead) {
	select {
	case rpc.reads <- read:
	case <-rpc.closed:
	}
}

func (rpc *appRPC) writeMessage(value any) error {
	data, err := json.Marshal(value)
	if err != nil || len(data) > maxRPCLine {
		return errors.New("app-server request exceeds JSONL limit")
	}
	rpc.write.Lock()
	defer rpc.write.Unlock()
	select {
	case <-rpc.closed:
		return errors.New("app-server RPC is closed")
	default:
	}
	data = append(data, '\n')
	for len(data) > 0 {
		n, writeErr := rpc.input.Write(data)
		if writeErr != nil {
			return errors.New("app-server request write failed")
		}
		if n == 0 {
			return errors.New("app-server request write made no progress")
		}
		data = data[n:]
	}
	return nil
}

func (rpc *appRPC) notify(method string, params any) error {
	return rpc.writeMessage(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (rpc *appRPC) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	stopOnCancel := context.AfterFunc(ctx, func() { _ = rpc.input.Close() })
	defer stopOnCancel()
	rpc.call.Lock()
	defer rpc.call.Unlock()
	rpc.nextID++
	id := rpc.nextID
	if err := rpc.writeMessage(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, &rpcRequestError{submitted: true, cause: err}
	}
	want, _ := json.Marshal(id)
	deadline := time.NewTimer(time.Until(deadlineFromContext(ctx)))
	defer deadline.Stop()
	for {
		select {
		case read := <-rpc.reads:
			if read.err != nil {
				return nil, &rpcRequestError{submitted: true, cause: read.err}
			}
			if string(read.message.ID) != string(want) {
				return nil, &rpcRequestError{submitted: true, cause: errors.New("unknown or out-of-order app-server response ID")}
			}
			if read.message.Error != nil {
				return nil, &rpcRequestError{cause: fmt.Errorf("app-server RPC %s failed: %d", method, read.message.Error.Code)}
			}
			return read.message.Result, nil
		case <-ctx.Done():
			return nil, &rpcRequestError{submitted: true, cause: ctx.Err()}
		case <-deadline.C:
			return nil, &rpcRequestError{submitted: true, cause: context.DeadlineExceeded}
		}
	}
}

func deadlineFromContext(ctx context.Context) time.Time {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline
	}
	return time.Now().Add(30 * time.Second)
}

func (rpc *appRPC) stop() {
	rpc.close.Do(func() {
		close(rpc.closed)
		_ = rpc.input.Close()
	})
}

func rpcParams(data any) json.RawMessage {
	encoded, _ := json.Marshal(data)
	return encoded
}

