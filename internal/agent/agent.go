package agent

import (
	"os"
	"path/filepath"
	"time"
)

type Info struct {
	Name   string
	Active bool
}

// knownAgents maps directory/file markers to agent names.
var knownAgents = map[string]string{
	".claude":                 "claude",
	".codex":                  "codex",
	".cursor":                 "cursor",
	".aider.chat.history.md":  "aider",
	".continue":               "continue",
}

const recentThreshold = 30 * time.Minute

// Detect checks a worktree path for known AI agent markers.
// Returns nil if no agent is detected.
func Detect(worktreePath string, agents []string) *Info {
	allowed := make(map[string]bool, len(agents))
	for _, a := range agents {
		allowed[a] = true
	}

	for marker, name := range knownAgents {
		if !allowed[name] {
			continue
		}

		fullPath := filepath.Join(worktreePath, marker)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		// For directories, check if recently modified
		if info.IsDir() {
			if isRecentDir(fullPath) {
				return &Info{Name: name, Active: true}
			}
			// Directory exists but not recent — still report it
			return &Info{Name: name, Active: false}
		}

		// For files, check mod time
		if time.Since(info.ModTime()) < recentThreshold {
			return &Info{Name: name, Active: true}
		}
		return &Info{Name: name, Active: false}
	}

	return nil
}

// isRecentDir checks if any file inside the directory was modified recently.
func isRecentDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < recentThreshold {
			return true
		}
	}
	return false
}
