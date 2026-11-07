package codex

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	codexClientID  = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexAuthURL   = "https://auth.openai.com/oauth/authorize"
	codexTokenURL  = "https://auth.openai.com/oauth/token"
	codexRedirect  = "http://localhost:1455/auth/callback"
	codexTokenFile = "openai-codex.auth"
)

const codexAccountClaim = "https://api.openai.com/auth"

type OpenAICodexCredential struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	ExpiresAt    int64  `json:"expires_at"`
}

type codexTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

var codexRefreshMu sync.Mutex

func OpenAICodexCredentialAvailable() bool {
	_, err := readOpenAICodexCredential(context.Background())
	return err == nil
}

// ResolveOpenAICodexCredential returns a current access token and serializes
// refresh-token rotation with credential-file reads and writes.
func ResolveOpenAICodexCredential(ctx context.Context) (OpenAICodexCredential, error) {
	codexRefreshMu.Lock()
	defer codexRefreshMu.Unlock()

	unlock, err := lockOpenAICodexCredential(ctx)
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	defer unlock()

	credential, err := readOpenAICodexCredentialUnlocked()
	if err != nil {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex is not logged in; run `aiw auth login openai-codex` or import a Codex login")
	}
	if credential.ExpiresAt > time.Now().Add(5*time.Minute).Unix() {
		return credential, nil
	}
	if credential.RefreshToken == "" {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex login expired; run `aiw auth login openai-codex` again")
	}

	refreshed, err := refreshOpenAICodexCredential(ctx, credential.RefreshToken)
	if err != nil {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex login could not be refreshed; run `aiw auth login openai-codex` again")
	}
	if err := writeOpenAICodexCredentialUnlocked(refreshed); err != nil {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex credentials refreshed but could not be saved")
	}
	return refreshed, nil
}

func OpenAICodexAuthStatus() (bool, time.Time, error) {
	credential, err := readOpenAICodexCredential(context.Background())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}
	return true, time.Unix(credential.ExpiresAt, 0).UTC(), nil
}

func LogoutOpenAICodex() error {
	unlock, err := lockOpenAICodexCredential(context.Background())
	if err != nil {
		return err
	}
	defer unlock()
	path, err := openAICodexCredentialPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ImportOpenAICodexCredential(ctx context.Context, source string) error {
	if strings.TrimSpace(source) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.New("cannot locate the user home directory")
		}
		if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
			source = filepath.Join(codexHome, "auth.json")
		} else {
			source = filepath.Join(home, ".codex", "auth.json")
		}
	}
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cannot read Codex auth file: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil { return errors.New("cannot read Codex auth file") }
	if len(data) > 1<<20 {
		return errors.New("Codex auth file exceeds 1 MiB")
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return errors.New("Codex auth file is not valid JSON")
	}
	var tokenRow map[string]json.RawMessage
	if raw := root["tokens"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &tokenRow)
	}
	if tokenRow == nil {
		tokenRow = root
	}
	var access, refresh string
	_ = json.Unmarshal(tokenRow["access_token"], &access)
	_ = json.Unmarshal(tokenRow["refresh_token"], &refresh)
	credential, err := credentialFromTokens(access, refresh, 0)
	if err != nil {
		return errors.New("Codex auth file does not contain importable ChatGPT access and refresh tokens")
	}
	unlock, err := lockOpenAICodexCredential(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	return writeOpenAICodexCredentialUnlocked(credential)
}

// LoginOpenAICodex starts the browser flow and stores the resulting credential.
// notify receives the URL to open. readInput is used only when the callback
// listener on 127.0.0.1:1455 cannot be started.
func LoginOpenAICodex(ctx context.Context, notify func(string), readInput func() (string, error)) error {
	verifierBytes := make([]byte, 32)
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(verifierBytes); err != nil {
		return errors.New("cannot create OAuth verifier")
	}
	if _, err := rand.Read(stateBytes); err != nil {
		return errors.New("cannot create OAuth state")
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	state := hex.EncodeToString(stateBytes)

	listener, listenErr := net.Listen("tcp", "127.0.0.1:1455")
	callback := make(chan string, 1)
	var server *http.Server
	if listenErr == nil {
		mux := http.NewServeMux()
		mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != state || r.URL.Query().Get("code") == "" {
				http.Error(w, "Invalid OAuth callback", http.StatusBadRequest)
				return
			}
			select {
			case callback <- r.URL.Query().Get("code"):
			default:
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, "AIW OpenAI Codex login completed. You may close this page.")
		})
		server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		go func() { _ = server.Serve(listener) }()
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(closeCtx)
		}()
	}

	authorize, _ := url.Parse(codexAuthURL)
	query := authorize.Query()
	query.Set("response_type", "code")
	query.Set("client_id", codexClientID)
	query.Set("redirect_uri", codexRedirect)
	query.Set("scope", "openid profile email offline_access")
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("state", state)
	query.Set("id_token_add_organizations", "true")
	query.Set("codex_cli_simplified_flow", "true")
	query.Set("originator", "aiw")
	authorize.RawQuery = query.Encode()
	if notify != nil {
		notify(authorize.String())
	}

	var code string
	if listenErr == nil {
		select {
		case code = <-callback:
		case <-ctx.Done():
			return errors.New("OpenAI Codex login cancelled")
		case <-time.After(15 * time.Minute):
			return errors.New("OpenAI Codex login timed out")
		}
	} else {
		if readInput == nil {
			return errors.New("cannot start OAuth callback listener and no manual input is available")
		}
		input, err := readInput()
		if err != nil {
			return err
		}
		parsed, err := url.Parse(strings.TrimSpace(input))
		if err != nil || parsed.Query().Get("state") != state {
			return errors.New("pasted OAuth callback URL has an invalid state")
		}
		code = parsed.Query().Get("code")
		if code == "" {
			return errors.New("pasted OAuth callback URL has no code")
		}
	}

	credential, err := exchangeOpenAICodexCode(ctx, code, verifier)
	if err != nil {
		return err
	}
	unlock, err := lockOpenAICodexCredential(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	return writeOpenAICodexCredentialUnlocked(credential)
}

func exchangeOpenAICodexCode(ctx context.Context, code, verifier string) (OpenAICodexCredential, error) {
	values := url.Values{
		"grant_type": {"authorization_code"},
		"client_id": {codexClientID},
		"code": {code},
		"code_verifier": {verifier},
		"redirect_uri": {codexRedirect},
	}
	token, err := requestOpenAICodexToken(ctx, values)
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	return credentialFromTokens(token.AccessToken, token.RefreshToken, token.ExpiresIn)
}

func refreshOpenAICodexCredential(ctx context.Context, refresh string) (OpenAICodexCredential, error) {
	values := url.Values{
		"grant_type": {"refresh_token"},
		"refresh_token": {refresh},
		"client_id": {codexClientID},
	}
	token, err := requestOpenAICodexToken(ctx, values)
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	return credentialFromTokens(token.AccessToken, token.RefreshToken, token.ExpiresIn)
}

func requestOpenAICodexToken(ctx context.Context, values url.Values) (codexTokenResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, codexTokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return codexTokenResponse{}, errors.New("cannot create OpenAI Codex token request")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return codexTokenResponse{}, errors.New("OpenAI Codex token request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return codexTokenResponse{}, fmt.Errorf("OpenAI Codex token request failed (HTTP %d)", response.StatusCode)
	}
	var token codexTokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&token); err != nil || token.AccessToken == "" || token.RefreshToken == "" || token.ExpiresIn <= 0 {
		return codexTokenResponse{}, errors.New("OpenAI Codex token response is invalid")
	}
	return token, nil
}

func credentialFromTokens(access, refresh string, expiresIn int64) (OpenAICodexCredential, error) {
	if access == "" || refresh == "" {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex token response is missing credentials")
	}
	claims, err := decodeJWTClaims(access)
	if err != nil {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex access token is invalid")
	}
	var auth struct {
		AccountID string `json:"chatgpt_account_id"`
	}
	if json.Unmarshal(claims[codexAccountClaim], &auth) != nil || auth.AccountID == "" {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex token has no account identifier")
	}
	expiresAt := int64(0)
	if raw := claims["exp"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &expiresAt)
	}
	if expiresAt == 0 && expiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second).Unix()
	}
	if expiresAt == 0 {
		return OpenAICodexCredential{}, errors.New("OpenAI Codex token expiry is missing")
	}
	return OpenAICodexCredential{AccessToken: access, RefreshToken: refresh, AccountID: auth.AccountID, ExpiresAt: expiresAt}, nil
}

func decodeJWTClaims(token string) (map[string]json.RawMessage, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(parts[1])
	}
	if err != nil {
		return nil, err
	}
	var claims map[string]json.RawMessage
	if err := json.Unmarshal(data, &claims); err != nil || claims == nil {
		return nil, errors.New("invalid JWT payload")
	}
	return claims, nil
}

func readOpenAICodexCredential(ctx context.Context) (OpenAICodexCredential, error) {
	unlock, err := lockOpenAICodexCredential(ctx)
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	defer unlock()
	return readOpenAICodexCredentialUnlocked()
}

func openAICodexCredentialPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", errors.New("cannot locate the user configuration directory")
	}
	return filepath.Join(root, "aiw", codexTokenFile), nil
}

func readOpenAICodexCredentialUnlocked() (OpenAICodexCredential, error) {
	path, err := openAICodexCredentialPath()
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	plain, err := decodeStoredOpenAICodexCredential(data)
	if err != nil {
		return OpenAICodexCredential{}, err
	}
	var credential OpenAICodexCredential
	if err := json.Unmarshal(plain, &credential); err != nil || credential.AccessToken == "" || credential.RefreshToken == "" || credential.AccountID == "" {
		return OpenAICodexCredential{}, errors.New("invalid stored OpenAI Codex credential")
	}
	return credential, nil
}

func writeOpenAICodexCredentialUnlocked(credential OpenAICodexCredential) error {
	path, err := openAICodexCredentialPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return errors.New("cannot create AIW credential directory")
	}
	plain, err := json.Marshal(credential)
	if err != nil {
		return errors.New("cannot encode OpenAI Codex credentials")
	}
	data, err := encodeStoredOpenAICodexCredential(plain)
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return errors.New("cannot write OpenAI Codex credentials")
	}
	if err := os.Chmod(temporary, 0o600); err != nil {
		_ = os.Remove(temporary)
		return errors.New("cannot restrict OpenAI Codex credential permissions")
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return errors.New("cannot save OpenAI Codex credentials")
	}
	return nil
}

