package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// expandStrict expands ${VAR} references using os.Expand and errors if any
// referenced variable is unset or empty in the process environment.
func expandStrict(s string) (string, error) {
	var missing string
	result := os.Expand(s, func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok || val == "" {
			missing = key
		}
		return val
	})
	if missing != "" {
		return "", fmt.Errorf("environment variable %q is unset or empty", missing)
	}
	return result, nil
}

// UpstreamConfig describes an external MCP server to proxy.
type UpstreamConfig struct {
	Name    string   `json:"name"`    // unique identifier, used as tool category and log prefix
	Command string   `json:"command"` // executable path (e.g. "node")
	Args    []string `json:"args"`    // command arguments
	Env     []string `json:"env"`     // environment variables for the subprocess
	Prefix  string   `json:"prefix"`  // required — tool names are prefixed: prefix_toolName
}

// LoadUpstreams reads upstream configs from a JSON file.
// Returns nil (no error) if the file does not exist or the path is empty.
func LoadUpstreams(path string) ([]UpstreamConfig, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is from trusted env var MCP_UPSTREAMS
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var configs []UpstreamConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}
	for i := range configs {
		if configs[i].Name == "" || configs[i].Command == "" {
			return nil, fmt.Errorf("upstream %d: name and command are required", i)
		}
		if configs[i].Prefix == "" {
			return nil, fmt.Errorf("upstream %q: prefix is required to avoid tool name collisions", configs[i].Name)
		}
		// Expand ${VAR} references in args and env from process environment
		// so secrets stay in .env, not in the JSON config file.
		// Errors if any referenced variable is unset or empty.
		for j, arg := range configs[i].Args {
			expanded, err := expandStrict(arg)
			if err != nil {
				return nil, fmt.Errorf("upstream %q: arg %d: %w", configs[i].Name, j, err)
			}
			configs[i].Args[j] = expanded
		}
		for j, env := range configs[i].Env {
			expanded, err := expandStrict(env)
			if err != nil {
				return nil, fmt.Errorf("upstream %q: env %d: %w", configs[i].Name, j, err)
			}
			configs[i].Env[j] = expanded
		}
	}
	return configs, nil
}
