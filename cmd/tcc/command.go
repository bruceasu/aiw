package tcc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
)

const defaultRoot = `c:\green\tcc`

func Dispatch(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printHelp()
		return nil
	}

	root := detectRoot()
	mode, rest := modeFor(args)
	exe := executable(root, is64BitMode(mode))
	return run(exe, argsFor(mode, rest, root)...)
}

func modeFor(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	switch strings.ToLower(args[0]) {
	case "dll", "run", "x86_64", "amd64", "x64":
		return strings.ToLower(args[0]), args[1:]
	default:
		return "", args
	}
}

func argsFor(mode string, args []string, root string) []string {
	result := make([]string, 0, len(args)+4)
	if mode == "dll" {
		result = append(result, "-shared")
	}
	if mode == "run" {
		result = append(result, "-run")
	}
	result = append(result, args...)
	if includeDir, ok := includeDir(root); ok {
		result = append(result, "-I"+includeDir)
	}
	if libDir, ok := libDir(root); ok {
		result = append(result, "-L"+libDir)
	}
	return result
}

func detectRoot() string {
	for _, key := range []string{"TCC_HOME", "TCC_ROOT", "TCC_DIR"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	if path, err := exec.LookPath("tcc.exe"); err == nil {
		return filepath.Dir(path)
	}
	if path, err := exec.LookPath("tcc"); err == nil {
		return filepath.Dir(path)
	}
	return defaultRoot
}

func executable(root string, prefer64 bool) string {
	candidates := []string{filepath.Join(root, "tcc.exe"), filepath.Join(root, "x86_64-win32-tcc.exe")}
	if prefer64 {
		candidates = []string{filepath.Join(root, "x86_64-win32-tcc.exe"), filepath.Join(root, "tcc.exe")}
	}
	for _, candidate := range candidates {
		if fsx.Exists(candidate) {
			return candidate
		}
	}
	if path, err := exec.LookPath("tcc.exe"); err == nil {
		return path
	}
	if path, err := exec.LookPath("tcc"); err == nil {
		return path
	}
	return filepath.Join(root, "tcc.exe")
}

func is64BitMode(mode string) bool {
	switch mode {
	case "x86_64", "amd64", "x64":
		return true
	default:
		return false
	}
}

func includeDir(root string) (string, bool) {
	path := filepath.Join(root, "include")
	if fsx.Exists(path) {
		return path, true
	}
	return "", false
}

func libDir(root string) (string, bool) {
	path := filepath.Join(root, "lib")
	if fsx.Exists(path) {
		return path, true
	}
	return "", false
}

func printHelp() {
	root := detectRoot()
	includePath, includeOK := includeDir(root)
	libPath, libOK := libDir(root)

	fmt.Println("Usage:")
	fmt.Println("  aiw tcc [args...]")
	fmt.Println("")
	fmt.Println("Modes:")
	fmt.Println("  tcc dll [args...]         Shortcut for -shared")
	fmt.Println("  tcc run [args...]         Shortcut for -run")
	fmt.Println("  tcc x86_64 [args...]      Use x86_64-win32-tcc.exe")
	fmt.Println("  default compiler: tcc.exe (32-bit)")
	fmt.Println("")
	fmt.Println("Auto paths:")
	fmt.Printf("  root: %s\n", root)
	fmt.Printf("  compiler: %s\n", filepath.Base(executable(root, false)))
	if includeOK {
		fmt.Printf("  include: %s\n", includePath)
	} else {
		fmt.Println("  include: <missing>")
	}
	if libOK {
		fmt.Printf("  lib: %s\n", libPath)
	} else {
		fmt.Println("  lib: <missing>")
	}
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  aiw tcc hello.c -o hello.exe")
	fmt.Println("  aiw tcc dll hello.c -o hello.dll")
	fmt.Println("  aiw tcc x86_64 hello.c -o hello.exe")
	fmt.Println("  aiw tcc run hello.c")
	fmt.Println("  aiw tcc hello.c -o app.exe -luser32")
}

func isHelpArg(arg string) bool {
	switch strings.ToLower(arg) {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
