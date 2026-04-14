package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Core       CoreConfig       `toml:"core"`
	Tmux       TmuxConfig       `toml:"tmux"`
	CI         CIConfig         `toml:"ci"`
	Agent      AgentConfig      `toml:"agent"`
	Hooks      HooksConfig      `toml:"hooks"`
	Appearance AppearanceConfig `toml:"appearance"`
}

type CoreConfig struct {
	BaseBranch     string `toml:"base_branch"`
	PathPattern    string `toml:"path_pattern"`
	AutoPruneAfter string `toml:"auto_prune_after"`
}

type TmuxConfig struct {
	Enabled       bool   `toml:"enabled"`
	AutoOpen      string `toml:"auto_open"`
	SessionPrefix string `toml:"session_prefix"`
	ShellCommand  string `toml:"shell_command"`
}

type CIConfig struct {
	Provider string `toml:"provider"`
}

type AgentConfig struct {
	Detect []string `toml:"detect"`
}

type HooksConfig struct {
	PostCreate []string `toml:"post_create"`
	PreDelete  []string `toml:"pre_delete"`
	PostDelete []string `toml:"post_delete"`
}

type AppearanceConfig struct {
	DefaultColor string `toml:"default_color"`
	DefaultIcon  string `toml:"default_icon"`
}

func Default() Config {
	return Config{
		Core: CoreConfig{
			PathPattern: "../{branch_slug}",
		},
		Tmux: TmuxConfig{
			Enabled:       true,
			AutoOpen:      "none",
			SessionPrefix: "grove",
		},
		CI: CIConfig{
			Provider: "auto",
		},
		Agent: AgentConfig{
			Detect: []string{"claude", "codex", "cursor", "aider"},
		},
		Appearance: AppearanceConfig{
			DefaultColor: "none",
		},
	}
}

func Load(repoRoot string) Config {
	cfg := Default()

	// Global config
	if home, err := os.UserHomeDir(); err == nil {
		toml.DecodeFile(filepath.Join(home, ".config", "grove", "config.toml"), &cfg)
	}

	// Repo-local overrides
	toml.DecodeFile(filepath.Join(repoRoot, ".grove.toml"), &cfg)

	return cfg
}
