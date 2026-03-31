package config

import (
	"encoding/json"
	"fmt"
	"os"
)

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
		for j, arg := range configs[i].Args {
			configs[i].Args[j] = os.ExpandEnv(arg)
		}
		for j, env := range configs[i].Env {
			configs[i].Env[j] = os.ExpandEnv(env)
		}
	}
	return configs, nil
}
