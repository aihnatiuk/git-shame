package git

import "time"

// BlameLine represents one line from git blame --porcelain output.
// Fields are denormalized for easy rendering (no pointer chasing in hot path).
type BlameLine struct {
	CommitHash  string // full 40-char SHA
	Author      string
	AuthorEmail string
	AuthorTime  time.Time
	Summary     string // first line of commit message
	LineNum     int    // final line number in the file (1-indexed)
	Content     string // raw line content (no trailing newline)
	Filename    string // "filename" field from porcelain (handles renames)
	Previous    PreviousCommit
}

// CommitMeta is used internally during parsing to accumulate per-commit data
// before it is copied into each BlameLine.
type CommitMeta struct {
	Hash        string
	Author      string
	AuthorEmail string
	AuthorTime  time.Time
	Summary     string
	Filename    string
	Previous    PreviousCommit
}

type PreviousCommit struct {
	Hash     string // full 40-char SHA of previous commit that touched this line (empty if none)
	Filename string // "filename" field from porcelain for previous commit (handles renames; empty if none)
}

// BlameResult is the message sent by RunBlameCmd to the Bubble Tea runtime.
type BlameResult struct {
	Lines []BlameLine
	Err   error
}
