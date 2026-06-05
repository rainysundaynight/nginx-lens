package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// installCompletion устанавливает shell completion для текущего shell.
func installCompletion() string {
	shell := detectShell()
	if shell == "" {
		return "Completion: определите shell вручную — nginx-lens completion bash"
	}
	path, manual, err := completionTarget(shell)
	if err != nil {
		return fmt.Sprintf("Completion не установлен: %v\n  Вручную: nginx-lens completion %s", err, shell)
	}
	if path == "" {
		return manual
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Sprintf("Completion не установлен в %s: %v\n  Вручную: nginx-lens completion %s > %s", path, err, shell, path)
	}
	defer f.Close()
	if err := writeCompletion(shell, f); err != nil {
		return fmt.Sprintf("Completion: %v", err)
	}
	msg := fmt.Sprintf("Completion установлен: %s (%s)", path, shell)
	if manual != "" {
		msg += "\n  " + manual
	}
	return msg
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		return ""
	}
	base := filepath.Base(sh)
	if strings.Contains(base, "zsh") {
		return "zsh"
	}
	if strings.Contains(base, "bash") {
		return "bash"
	}
	if strings.Contains(base, "fish") {
		return "fish"
	}
	return ""
}

func completionTarget(shell string) (path string, manual string, err error) {
	home, _ := os.UserHomeDir()
	switch shell {
	case "bash":
		if _, e := os.Stat("/etc/bash_completion.d"); e == nil && os.Geteuid() == 0 {
			return "/etc/bash_completion.d/nginx-lens", "", nil
		}
		dir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", "", err
		}
		return filepath.Join(dir, "nginx-lens"), "", nil
	case "zsh":
		dir := filepath.Join(home, ".zsh", "completions")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", "", err
		}
		p := filepath.Join(dir, "_nginx-lens")
		hint := "Добавьте в ~/.zshrc: fpath=(~/.zsh/completions $fpath); autoload -Uz compinit && compinit"
		return p, hint, nil
	case "fish":
		dir := filepath.Join(home, ".config", "fish", "completions")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", "", err
		}
		return filepath.Join(dir, "nginx-lens.fish"), "", nil
	case "powershell":
		return "", "PowerShell: nginx-lens completion powershell | Out-File $PROFILE -Append", nil
	default:
		return "", "", fmt.Errorf("неподдерживаемый shell: %s", shell)
	}
}

func writeCompletion(shell string, w *os.File) error {
	switch shell {
	case "bash":
		return NewRoot().GenBashCompletion(w)
	case "zsh":
		return NewRoot().GenZshCompletion(w)
	case "fish":
		return NewRoot().GenFishCompletion(w, true)
	default:
		return fmt.Errorf("shell %s", shell)
	}
}
