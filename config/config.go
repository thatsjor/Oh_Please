package config

import (
	"bufio"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type Config struct {
	Keys map[string][]string
}

func NewConfig() *Config {
	return &Config{
		Keys: map[string][]string{
			"save":          {"ctrl+s"},
			"close_tab":     {"ctrl+w"},
			"next_tab":      {"ctrl+f", "alt+right"},
			"prev_tab":      {"ctrl+b", "alt+left"},
			"quit":          {"ctrl+q", "ctrl+c", "esc"},
			"switch_focus":  {"tab"},
			"up":            {"up", "k"},
			"down":          {"down", "j"},
			"enter":         {"enter"},
			"toggle_hidden": {"."},
			"toggle_zen":    {"ctrl+z"},
			"new_file":      {"ctrl+n"},
		},
	}
}

func GetUserHome() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		if u, err := user.Lookup(sudoUser); err == nil {
			return u.HomeDir
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

func LoadConfig() *Config {
	c := NewConfig()

	home := GetUserHome()
	configPath := filepath.Join(home, ".config", "opls", "config.conf")

	if _, err := os.Stat(configPath); err == nil {
		c.ParseINI(configPath)
	}

	return c
}

func (c *Config) IsAction(action string, key string) bool {
	keys, ok := c.Keys[action]
	if !ok {
		return false
	}
	for _, k := range keys {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

func (c *Config) ParseINI(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			val := strings.TrimSpace(parts[1])

			switch currentSection {
			case "keys":
				rawKeys := strings.Split(val, ",")
				var cleanKeys []string
				for _, rk := range rawKeys {
					cleanKeys = append(cleanKeys, strings.TrimSpace(rk))
				}
				c.Keys[key] = cleanKeys
			}
		}
	}
}
