// Package dotenv loads KEY=VALUE lines from a file into the process environment
// without overriding variables already set in the environment.
package dotenv

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load reads path line-by-line and calls os.Setenv(k, v) only when os.LookupEnv(k)
// reports the key is not already set (common dotenv behavior).
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("dotenv: open %s: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("dotenv: %s:%d: expected KEY=VALUE", path, lineNum)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("dotenv: %s:%d: empty key", path, lineNum)
		}
		val = unquote(strings.TrimSpace(val))
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("dotenv: set %s: %w", key, err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("dotenv: read %s: %w", path, err)
	}
	return nil
}

func unquote(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			return s[1 : len(s)-1]
		}
	}
	return s
}
