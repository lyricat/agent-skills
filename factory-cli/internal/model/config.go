package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var prefixRe = regexp.MustCompile(`^[A-Z]+$`)

// Config holds settings from .factory/factory.toml with defaults aligned to factory/SKILL.md.
type Config struct {
	Base        string   `toml:"base" json:"base"`
	Remote      string   `toml:"remote" json:"remote"`
	Owners      []string `toml:"owners" json:"owners"`
	Test        string   `toml:"test" json:"test"`
	AfterChange []string `toml:"after_change" json:"after_change"`
	Read        []string `toml:"read" json:"read"`
	Never       []string `toml:"never" json:"never"`
	Rules       []string `toml:"rules" json:"rules"`
	MaxRounds   int      `toml:"max_rounds" json:"max_rounds"`
	Ship        []string `toml:"ship" json:"ship"`
	Prefix      string   `toml:"prefix" json:"prefix"`
	Board       string   `toml:"board" json:"board"`
	Timezone    string   `toml:"timezone" json:"timezone"`
}

// DefaultConfig returns a Config populated with the factory defaults.
func DefaultConfig() Config {
	return Config{
		Base:      "master",
		Remote:    "origin",
		MaxRounds: 1,
		Ship:      []string{"merge", "tag", "push"},
		Prefix:    "FT",
		Board:     "web",
	}
}

// ParseConfig decodes factory.toml bytes and resolves the configured timezone.
func ParseConfig(data []byte) (Config, *time.Location, error) {
	cfg := DefaultConfig()
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, time.Local, fmt.Errorf("parse factory.toml: %w", err)
	}
	if strings.TrimSpace(cfg.Base) == "" {
		cfg.Base = "master"
	}
	if strings.TrimSpace(cfg.Remote) == "" {
		cfg.Remote = "origin"
	}
	if cfg.MaxRounds <= 0 && !md.IsDefined("max_rounds") {
		cfg.MaxRounds = 1
	}
	if !md.IsDefined("ship") {
		cfg.Ship = []string{"merge", "tag", "push"}
	}
	if strings.TrimSpace(cfg.Prefix) == "" {
		cfg.Prefix = "FT"
	}
	if !prefixRe.MatchString(cfg.Prefix) {
		return cfg, time.Local, fmt.Errorf("factory.toml: prefix must be capital letters, not %q", cfg.Prefix)
	}
	if strings.TrimSpace(cfg.Board) == "" {
		cfg.Board = "web"
	}

	loc := time.Local
	if tz := strings.TrimSpace(cfg.Timezone); tz != "" {
		loaded, err := time.LoadLocation(tz)
		if err != nil {
			return cfg, time.Local, fmt.Errorf("factory.toml: invalid timezone %q: %w", tz, err)
		}
		loc = loaded
	}
	return cfg, loc, nil
}

// SignerSet returns the set of recognized signers (owners plus "factory").
func (c Config) SignerSet() map[string]bool {
	s := map[string]bool{"factory": true}
	for _, owner := range c.Owners {
		if o := strings.TrimSpace(owner); o != "" {
			s[o] = true
		}
	}
	return s
}

// OwnerSet returns the set of owner names from config.
func (c Config) OwnerSet() map[string]bool {
	s := make(map[string]bool, len(c.Owners))
	for _, owner := range c.Owners {
		if o := strings.TrimSpace(owner); o != "" {
			s[o] = true
		}
	}
	return s
}
