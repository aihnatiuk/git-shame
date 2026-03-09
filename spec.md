# Project Spec: "shame" - A Modern Git Blame TUI

## Overview
"shame" is a high-performance terminal UI tool written in Go for interactive git blame exploration. It is intended to be a modern, user-friendly alternative to `tig blame`.

## Core Tech Stack
- **Language:** Go
- **TUI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea/blob/main/README.md) / [Bubbles](https://github.com/charmbracelet/bubbles/blob/main/README.md)
- **Syntax Highlighting:** [Chroma](https://github.com/alecthomas/chroma)
- **Git Integration:** Git CLI wrapper (preferred for speed).

## Key Features
- **Interactive Blame:** Navigate file history by jumping to the commit that last changed a specific line.
The tool mimics the behavior of `tig blame` allowing to navigate to the parent/child commit chat changed a particular line.
- **Vim-like UX:** `j/k` for navigation, `ctrl-d` for page down, `ctrl-u` for page up, `/` for search.
- **Dynamic Columns:** Toggleable columns for Hash, Date, Author, Message, Line Number, Code and other columns that git provides in the blame output.
- **Configuration:** Support for a config file (YAML) to define custom keybindings and syntax themes.
- **Performance:** Non-blocking UI; the blame data should not freeze the terminal during heavy computation.

## Initial CLI Arguments
- `shame <file>`: Blame the file at HEAD.
- `shame <file> <revision>`: Blame the file at a specific commit/branch.

## Implementation notes
- Follow a modular, loosely coupled design.
- Use Go's best practices.
