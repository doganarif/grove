package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Worktree struct {
	Path       string
	Branch     string
	Head       string
	IsMain     bool
	IsDetached bool
	Locked     bool
	Prunable   bool
}

type Status struct {
	Modified  int
	Added     int
	Deleted   int
	Untracked int
}

func (s Status) Clean() bool {
	return s.Modified == 0 && s.Added == 0 && s.Deleted == 0 && s.Untracked == 0
}

func (s Status) String() string {
	if s.Clean() {
		return "clean"
	}
	var parts []string
	if s.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%dM", s.Modified))
	}
	if s.Added > 0 {
		parts = append(parts, fmt.Sprintf("%dA", s.Added))
	}
	if s.Deleted > 0 {
		parts = append(parts, fmt.Sprintf("%dD", s.Deleted))
	}
	if s.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%dU", s.Untracked))
	}
	return strings.Join(parts, " ")
}

type RemoteStatus struct {
	Ahead    int
	Behind   int
	Gone     bool
	NoRemote bool
}

func (r RemoteStatus) String() string {
	if r.Gone {
		return "gone"
	}
	if r.NoRemote {
		return "—"
	}
	if r.Ahead == 0 && r.Behind == 0 {
		return "✓"
	}
	var parts []string
	if r.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", r.Ahead))
	}
	if r.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", r.Behind))
	}
	return strings.Join(parts, "")
}

type WorktreeInfo struct {
	Status   Status
	Remote   RemoteStatus
	LastSHA  string
	LastMsg  string
	LastTime time.Time
	Files    []string
}

// run executes a git command and returns trimmed stdout.
func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func runInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git -C %s %s: %s", dir, strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func RepoRoot() (string, error) {
	bare, _ := run("rev-parse", "--is-bare-repository")
	if bare == "true" {
		gitDir, err := run("rev-parse", "--git-dir")
		if err != nil {
			return "", fmt.Errorf("not a git repository")
		}
		abs, err := filepath.Abs(gitDir)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	toplevel, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return toplevel, nil
}

func IsBareRepo() bool {
	out, err := run("rev-parse", "--is-bare-repository")
	return err == nil && out == "true"
}

func BaseBranch() string {
	if _, err := run("rev-parse", "--verify", "refs/heads/main"); err == nil {
		return "main"
	}
	if _, err := run("rev-parse", "--verify", "refs/heads/master"); err == nil {
		return "master"
	}
	return "main"
}

func ListWorktrees() ([]Worktree, error) {
	out, err := run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current *Worktree
	isFirst := true

	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{
				Path:   strings.TrimPrefix(line, "worktree "),
				IsMain: isFirst,
			}
			isFirst = false
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		} else if line == "detached" && current != nil {
			current.IsDetached = true
		} else if strings.HasPrefix(line, "locked") && current != nil {
			current.Locked = true
		} else if line == "prunable" && current != nil {
			current.Prunable = true
		}
	}
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

func WorktreeStatus(path string) (Status, error) {
	out, err := runInDir(path, "status", "--porcelain")
	if err != nil {
		return Status{}, err
	}
	if out == "" {
		return Status{}, nil
	}

	var s Status
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		switch {
		case x == '?' && y == '?':
			s.Untracked++
		case x == 'A' || y == 'A':
			s.Added++
		case x == 'D' || y == 'D':
			s.Deleted++
		default:
			s.Modified++
		}
	}
	return s, nil
}

func RemoteTracking(path string) RemoteStatus {
	branch, err := runInDir(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "HEAD" {
		return RemoteStatus{NoRemote: true}
	}

	// Check if upstream exists
	_, err = runInDir(path, "rev-parse", "--abbrev-ref", branch+"@{u}")
	if err != nil {
		// Check for gone tracking branch
		trackInfo, _ := runInDir(path, "for-each-ref", "--format=%(upstream:track)", "refs/heads/"+branch)
		if strings.Contains(trackInfo, "gone") {
			return RemoteStatus{Gone: true}
		}
		return RemoteStatus{NoRemote: true}
	}

	out, err := runInDir(path, "rev-list", "--left-right", "--count", branch+"@{u}...HEAD")
	if err != nil {
		return RemoteStatus{NoRemote: true}
	}

	parts := strings.Fields(out)
	if len(parts) == 2 {
		behind, _ := strconv.Atoi(parts[0])
		ahead, _ := strconv.Atoi(parts[1])
		return RemoteStatus{Ahead: ahead, Behind: behind}
	}
	return RemoteStatus{NoRemote: true}
}

func LastCommit(path string) (sha, message string, when time.Time, err error) {
	out, err := runInDir(path, "log", "-1", "--format=%h|%s|%ct")
	if err != nil {
		return "", "", time.Time{}, err
	}
	parts := strings.SplitN(out, "|", 3)
	if len(parts) < 3 {
		return "", "", time.Time{}, fmt.Errorf("unexpected log format")
	}
	ts, _ := strconv.ParseInt(parts[2], 10, 64)
	return parts[0], parts[1], time.Unix(ts, 0), nil
}

func ListBranches() ([]string, error) {
	local, err := run("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	remote, _ := run("branch", "-r", "--format=%(refname:short)")

	var branches []string
	seen := make(map[string]bool)

	for _, b := range strings.Split(local, "\n") {
		b = strings.TrimSpace(b)
		if b != "" && !seen[b] {
			branches = append(branches, b)
			seen[b] = true
		}
	}
	for _, b := range strings.Split(remote, "\n") {
		b = strings.TrimSpace(b)
		short := strings.TrimPrefix(b, "origin/")
		if b != "" && !seen[short] && short != "HEAD" {
			branches = append(branches, short)
			seen[short] = true
		}
	}

	return branches, nil
}

func AddWorktree(path, branch, base string, createBranch bool) error {
	if createBranch {
		_, err := run("worktree", "add", "-b", branch, path, base)
		return err
	}
	_, err := run("worktree", "add", path, branch)
	return err
}

func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := run(args...)
	return err
}

func DeleteBranch(name string) error {
	_, err := run("branch", "-D", name)
	return err
}

func BranchExists(name string) bool {
	_, err := run("rev-parse", "--verify", "refs/heads/"+name)
	return err == nil
}

func IsMerged(branch, base string) bool {
	out, err := run("branch", "--merged", base)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		b := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "* "))
		if b == branch {
			return true
		}
	}
	return false
}

func ChangedFiles(path string) []string {
	out, err := runInDir(path, "status", "--porcelain")
	if err != nil || out == "" {
		return nil
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) > 2 {
			files = append(files, line)
		}
	}
	return files
}

func ParallelLoadInfo(worktrees []Worktree) []WorktreeInfo {
	infos := make([]WorktreeInfo, len(worktrees))
	var wg sync.WaitGroup

	for i, wt := range worktrees {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			infos[i].Status, _ = WorktreeStatus(path)
			infos[i].Remote = RemoteTracking(path)
			infos[i].LastSHA, infos[i].LastMsg, infos[i].LastTime, _ = LastCommit(path)
			infos[i].Files = ChangedFiles(path)
		}(i, wt.Path)
	}

	wg.Wait()
	return infos
}

func CurrentWorktreePath() string {
	toplevel, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return toplevel
}

func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}
