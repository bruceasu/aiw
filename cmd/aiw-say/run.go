package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"aiw/internal/say"
	"aiw/internal/version"
)

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	for _, arg := range args {
		name := strings.SplitN(arg, "=", 2)[0]
		switch name {
		case "--clipboard", "--copy", "--file", "--output", "-o", "--dialog", "--pair":
			return fmt.Errorf("option %s is not implemented", name)
		}
	}
	var diagnostics strings.Builder
	opts, err := parseOptions(args, &diagnostics)
	if errors.Is(err, flag.ErrHelp) || opts.help {
		usage(stdout)
		return nil
	}
	if err != nil {
		return optionsError(err)
	}
	if opts.version {
		_, err := fmt.Fprintln(stdout, "aiw-say "+version.Label())
		return err
	}
	if len(opts.args) > 0 && stdinIsActive(stdin) {
		return errors.New("provide text as an argument or stdin, not both")
	}
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	cfg, err := say.LoadBase(exePath, opts.config)
	if err != nil {
		return err
	}
	cfg, err = say.LoadProfile(cfg, opts.profile)
	if err != nil {
		return err
	}
	applyOptions(&cfg, opts)
	duration, err := say.Validate(cfg)
	if err != nil {
		return err
	}
	text, err := say.ReadInput(opts.args, stdin)
	if err != nil {
		return err
	}
	request := say.Request{
		Source: cfg.Source, Target: cfg.Target, Mode: cfg.Mode, Style: cfg.Style,
		Polite: cfg.Polite, Simple: cfg.Simple, Profanity: cfg.Profanity,
		Text: text, Model: cfg.Model,
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	translation, err := say.NewOpenAIProvider().Translate(ctx, request)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, translation); err != nil {
		return fmt.Errorf("write translation: %w", err)
	}
	return nil
}

func stdinIsActive(stdin io.Reader) bool {
	if stdin == nil { return false }
	file, ok := stdin.(*os.File)
	if !ok { return true }
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice == 0
}

func applyOptions(cfg *say.Config, opts cliOptions) {
	if opts.target != "" { cfg.Target = opts.target }
	if opts.source != "" { cfg.Source = opts.source }
	if opts.mode != "" { cfg.Mode = opts.mode }
	if opts.style != "" { cfg.Style = opts.style }
	if opts.polite != "" { cfg.Polite = opts.polite }
	if opts.profanity != "" { cfg.Profanity = opts.profanity }
	if opts.provider != "" { cfg.Provider = opts.provider }
	if opts.model != "" { cfg.Model = opts.model }
	if opts.timeout != "" { cfg.Timeout = opts.timeout }
	if opts.simpleSet { cfg.Simple = opts.simple }
}
