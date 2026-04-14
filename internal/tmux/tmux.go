package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Multiplexer int

const (
	None Multiplexer = iota
	Tmux
	Zellij
)

// Detect returns which terminal multiplexer is active.
func Detect() Multiplexer {
	if os.Getenv("TMUX") != "" {
		return Tmux
	}
	if os.Getenv("ZELLIJ") != "" {
		return Zellij
	}
	return None
}

func (m Multiplexer) String() string {
	switch m {
	case Tmux:
		return "tmux"
	case Zellij:
		return "zellij"
	default:
		return "none"
	}
}

// OpenWindow opens a new window/tab in the detected multiplexer.
func OpenWindow(path, name, shellCmd string) error {
	switch Detect() {
	case Tmux:
		return tmuxRun("new-window", "-c", path, "-n", name)
	case Zellij:
		args := []string{"action", "new-tab", "--cwd", path, "--name", name}
		return zellijRun(args...)
	default:
		return fmt.Errorf("no multiplexer detected")
	}
}

// OpenHSplit opens a horizontal split.
func OpenHSplit(path, shellCmd string) error {
	switch Detect() {
	case Tmux:
		return tmuxRun("split-window", "-h", "-c", path)
	case Zellij:
		return zellijRun("action", "new-pane", "--direction", "right", "--cwd", path)
	default:
		return fmt.Errorf("no multiplexer detected")
	}
}

// OpenVSplit opens a vertical split.
func OpenVSplit(path, shellCmd string) error {
	switch Detect() {
	case Tmux:
		return tmuxRun("split-window", "-v", "-c", path)
	case Zellij:
		return zellijRun("action", "new-pane", "--direction", "down", "--cwd", path)
	default:
		return fmt.Errorf("no multiplexer detected")
	}
}

// SessionExists checks if a tmux session with the given name exists.
func SessionExists(name string) bool {
	if Detect() != Tmux {
		return false
	}
	err := exec.Command("tmux", "has-session", "-t", name).Run()
	return err == nil
}

// KillSession kills a tmux session by name.
func KillSession(name string) error {
	return tmuxRun("kill-session", "-t", name)
}

// ListSessions returns names of active tmux sessions.
func ListSessions() []string {
	if Detect() != Tmux {
		return nil
	}
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil
	}
	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions
}

func tmuxRun(args ...string) error {
	return exec.Command("tmux", args...).Run()
}

func zellijRun(args ...string) error {
	return exec.Command("zellij", args...).Run()
}
