package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/strubio-ray/vm-ward/tui/internal/model"
	"github.com/strubio-ray/vm-ward/tui/internal/vmw"
)

func main() {
	vmwPath, err := resolveVmwPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	client := &vmw.ExecClient{VmwPath: vmwPath}
	m := model.New(client)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolveVmwPath finds the vmw bash script.
// Priority: VMW_PATH env var > sibling binary > PATH lookup.
func resolveVmwPath() (string, error) {
	// 1. Explicit env var (set by `vmw tui` dispatch)
	if p := os.Getenv("VMW_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. Sibling of this binary
	self, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(self), "vmw")
		if _, err := os.Stat(sibling); err == nil {
			return sibling, nil
		}
	}

	// 3. PATH lookup
	if p, err := exec.LookPath("vmw"); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("vmw not found. Set VMW_PATH or ensure vmw is in your PATH")
}
