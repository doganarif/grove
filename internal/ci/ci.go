package ci

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Status struct {
	State    string // "passed", "failed", "running", "cancelled", "none"
	Provider string // "github", "gitlab"
	RunName  string
	RunURL   string
}

type ghRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Name       string `json:"name"`
	HTMLURL    string `json:"url"`
}

// DetectProvider guesses the CI provider from the git remote URL.
func DetectProvider() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "none"
	}
	url := strings.TrimSpace(string(out))
	switch {
	case strings.Contains(url, "github.com"):
		return "github"
	case strings.Contains(url, "gitlab"):
		return "gitlab"
	default:
		return "none"
	}
}

// FetchAll loads CI status for multiple branches concurrently.
func FetchAll(branches []string, provider string) map[string]Status {
	if provider == "auto" {
		provider = DetectProvider()
	}
	if provider == "none" {
		return nil
	}

	results := make(map[string]Status)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, branch := range branches {
		wg.Add(1)
		go func(b string) {
			defer wg.Done()
			s := fetch(b, provider)
			mu.Lock()
			results[b] = s
			mu.Unlock()
		}(branch)
	}

	wg.Wait()
	return results
}

func fetch(branch, provider string) Status {
	switch provider {
	case "github":
		return fetchGitHub(branch)
	case "gitlab":
		return fetchGitLab(branch)
	default:
		return Status{State: "none"}
	}
}

func fetchGitHub(branch string) Status {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "run", "list",
		"--branch", branch,
		"--limit", "1",
		"--json", "status,conclusion,name,url",
	)
	out, err := cmd.Output()
	if err != nil {
		return Status{State: "none", Provider: "github"}
	}

	var runs []ghRun
	if err := json.Unmarshal(out, &runs); err != nil || len(runs) == 0 {
		return Status{State: "none", Provider: "github"}
	}

	run := runs[0]
	s := Status{
		Provider: "github",
		RunName:  run.Name,
		RunURL:   run.HTMLURL,
	}

	switch run.Status {
	case "completed":
		switch run.Conclusion {
		case "success":
			s.State = "passed"
		case "failure":
			s.State = "failed"
		case "cancelled":
			s.State = "cancelled"
		default:
			s.State = "failed"
		}
	case "in_progress", "queued", "waiting":
		s.State = "running"
	default:
		s.State = "none"
	}

	return s
}

func fetchGitLab(branch string) Status {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "glab", "ci", "list",
		"--branch", branch,
		"--per-page", "1",
		"--output", "json",
	)
	out, err := cmd.Output()
	if err != nil {
		return Status{State: "none", Provider: "gitlab"}
	}

	var pipelines []struct {
		Status string `json:"status"`
		WebURL string `json:"web_url"`
	}
	if err := json.Unmarshal(out, &pipelines); err != nil || len(pipelines) == 0 {
		return Status{State: "none", Provider: "gitlab"}
	}

	p := pipelines[0]
	s := Status{Provider: "gitlab", RunURL: p.WebURL}

	switch p.Status {
	case "success":
		s.State = "passed"
	case "failed":
		s.State = "failed"
	case "running", "pending", "created":
		s.State = "running"
	case "canceled", "skipped":
		s.State = "cancelled"
	default:
		s.State = "none"
	}

	return s
}
