package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	codex "aiw/pkg/openai-codex"
)

func Dispatch(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("Usage: aiw auth login openai-codex | import-codex [--from PATH] | status | logout")
		return nil
	}
	switch args[0] {
	case "login":
		if len(args) != 2 || args[1] != "openai-codex" { return errors.New("usage: aiw auth login openai-codex") }
		err := codex.LoginOpenAICodex(context.Background(), func(url string) {
			fmt.Println("Open this URL in your browser to sign in:")
			fmt.Println(url)
			fmt.Println("AIW will finish after the browser redirects to the local callback.")
		}, func() (string, error) {
			fmt.Print("Callback listener unavailable. Paste the full redirect URL: ")
			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			return strings.TrimSpace(line), err
		})
		if err != nil { return err }
		fmt.Println("OpenAI Codex credentials saved for this user.")
		return nil
	case "import-codex":
		var source string
		if len(args) == 3 && args[1] == "--from" { source = args[2] } else if len(args) != 1 { return errors.New("usage: aiw auth import-codex [--from PATH]") }
		if err := codex.ImportOpenAICodexCredential(context.Background(), source); err != nil { return err }
		fmt.Println("Codex credentials imported into AIW.")
		return nil
	case "status":
		if len(args) != 1 { return errors.New("usage: aiw auth status") }
		present, expires, err := codex.OpenAICodexAuthStatus()
		if err != nil { return err }
		if !present { fmt.Println("OpenAI Codex: not logged in"); return nil }
		if expires.Before(time.Now()) { fmt.Printf("OpenAI Codex: login expired at %s; refresh may still recover it\n", expires.Format(time.RFC3339)) } else { fmt.Printf("OpenAI Codex: logged in; access token expires at %s\n", expires.Format(time.RFC3339)) }
		return nil
	case "logout":
		if len(args) != 1 { return errors.New("usage: aiw auth logout") }
		if err := codex.LogoutOpenAICodex(); err != nil { return err }
		fmt.Println("OpenAI Codex credentials removed from AIW.")
		return nil
	default:
		return errors.New("unknown auth command; use `aiw auth --help`")
	}
}
