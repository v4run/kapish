// Package config defines kapish's configuration types, defaults, and the
// load/validate/persist pipeline. The merge order is:
//
//	built-in Defaults() < config file < env vars < command-line flags
package config

// Config is the top-level kapish configuration.
type Config struct {
	ManagementClusters ManagementClustersConfig `yaml:"managementClusters"`
	Shell              ShellConfig              `yaml:"shell"`
	UI                 UIConfig                 `yaml:"ui"`
	Web                WebConfig                `yaml:"web"`
}

type ManagementClustersConfig struct {
	Current string                   `yaml:"current,omitempty"`
	Entries []ManagementClusterEntry `yaml:"entries,omitempty"`
}

type ManagementClusterEntry struct {
	Name       string `yaml:"name"`
	Kubeconfig string `yaml:"kubeconfig,omitempty"`
	Context    string `yaml:"context,omitempty"`
	Namespace  string `yaml:"namespace,omitempty"`
}

type ShellConfig struct {
	Command string            `yaml:"command,omitempty"`
	Cwd     string            `yaml:"cwd,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty"`
	Prompt  string            `yaml:"prompt,omitempty"`
}

type UIConfig struct {
	Theme              string `yaml:"theme"`
	RefreshIntervalSec int    `yaml:"refreshIntervalSec"`
	OneShot            bool   `yaml:"oneShot"`
}

type WebConfig struct {
	DefaultPort int    `yaml:"defaultPort"`
	OpenBrowser bool   `yaml:"openBrowser"`
	BindAddr    string `yaml:"bindAddr"`
}

// Defaults returns a fresh built-in Config. Each call returns an
// independent copy — callers can mutate the result freely.
func Defaults() Config {
	return Config{
		ManagementClusters: ManagementClustersConfig{
			Current: "",
			Entries: nil,
		},
		Shell: ShellConfig{
			Command: "",
			Cwd:     "",
			Env:     map[string]string{},
			Aliases: map[string]string{},
			Prompt:  "[{cluster}] ",
		},
		UI: UIConfig{
			Theme:              "dark",
			RefreshIntervalSec: 30,
			OneShot:            false,
		},
		Web: WebConfig{
			DefaultPort: 0,
			OpenBrowser: true,
			BindAddr:    "127.0.0.1",
		},
	}
}
