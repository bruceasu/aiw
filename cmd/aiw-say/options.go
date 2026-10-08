package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

type cliOptions struct {
	args []string
	target, source, mode, style, polite, profanity string
	provider, model, timeout, config, profile string
	simple, simpleSet, help, version bool
}

func parseOptions(args []string, output io.Writer) (cliOptions, error) {
	var o cliOptions
	fs := flag.NewFlagSet("aiw say", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&o.target, "target", "", "target language: zh, ja, or en")
	fs.StringVar(&o.target, "t", "", "target language: zh, ja, or en")
	fs.StringVar(&o.source, "source", "", "source language: auto, zh, ja, or en")
	fs.StringVar(&o.source, "s", "", "source language: auto, zh, ja, or en")
	fs.StringVar(&o.mode, "mode", "", "translation mode: realtime or written")
	fs.StringVar(&o.style, "style", "", "translation style")
	fs.StringVar(&o.polite, "polite", "", "politeness: casual, polite, or formal")
	fs.StringVar(&o.polite, "p", "", "politeness: casual, polite, or formal")
	fs.BoolVar(&o.simple, "simple", false, "simplify the translation")
	fs.StringVar(&o.profanity, "profanity", "", "profanity handling: mask, soften, or preserve")
	fs.StringVar(&o.profile, "profile", "", "user profile name")
	fs.StringVar(&o.provider, "provider", "", "translation provider")
	fs.StringVar(&o.model, "model", "", "configured model name")
	fs.StringVar(&o.timeout, "timeout", "", "request timeout (for example 30s)")
	fs.StringVar(&o.config, "config", "", "base TOML configuration path")
	fs.BoolVar(&o.help, "help", false, "show help")
	fs.BoolVar(&o.version, "version", false, "show version")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) { return o, flag.ErrHelp }
		return o, err
	}
	o.args = fs.Args()
	fs.Visit(func(f *flag.Flag) { if f.Name == "simple" { o.simpleSet = true } })
	return o, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `Usage: aiw say [options] [text]

Translate one text argument or UTF-8 text from stdin. On success, stdout contains only the translation.

Options:
  -t, --target LANG       Target language (default ja)
  -s, --source LANG       Source language (default auto)
      --mode MODE         realtime or written
      --style STYLE       spoken, teams, letter, document, or article
  -p, --polite LEVEL      casual, polite, or formal
      --simple            Simplify the translation
      --profanity MODE    mask, soften, or preserve
      --profile NAME      Read a user profile
      --provider NAME     Translation provider (openai)
      --model NAME        Explicit configured model
      --timeout DURATION  Request timeout (default 30s)
      --config PATH       Base TOML configuration
      --help              Show this help
      --version           Show version

Clipboard, file, output-file, dialog, and pair modes are not implemented.`)
}

func optionsError(err error) error {
	message := strings.TrimSpace(err.Error())
	if message == "" { return errors.New("invalid options") }
	return fmt.Errorf("%s", message)
}
