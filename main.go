package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "shame: "+err.Error())
			os.Exit(1)
		}
		defer f.Close()
		log.SetOutput(f)
		log.SetPrefix("shame ")
	} else {
		log.SetOutput(io.Discard)
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: shame <file> [revision]")
		os.Exit(1)
	}

	displayFile := os.Args[1]
	revision := ""
	if len(os.Args) >= 3 {
		revision = os.Args[2]
	}

	// Resolve to absolute path so all git operations work regardless of cwd.
	absFile, err := filepath.Abs(displayFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shame: "+err.Error())
		os.Exit(1)
	}

	// Find the git repository root.
	repoRoot, err := git.RepoRoot(absFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shame: "+err.Error())
		os.Exit(1)
	}

	// Compute the file path relative to the repo root (required by git blame).
	relFile, err := git.RelPath(repoRoot, absFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shame: "+err.Error())
		os.Exit(1)
	}

	app := ui.NewApp(repoRoot, relFile, displayFile, revision)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "shame: "+err.Error())
		os.Exit(1)
	}
}
