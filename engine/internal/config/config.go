package config

import (
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	MongoDB  MongoDBConfig  `yaml:"mongodb"`
	Game     GameConfig     `yaml:"game"`
	Auth     AuthConfig     `yaml:"auth"`
	Email    EmailConfig    `yaml:"email"`
	Feedback FeedbackConfig `yaml:"feedback"`
}

type FeedbackConfig struct {
	VibectlURL    string `yaml:"vibectl_url"`
	VibectlAPIKey string `yaml:"vibectl_api_key"`
}

type AuthConfig struct {
	GoogleClientID string `yaml:"google_client_id"`
	JWTSecret      string `yaml:"jwt_secret"`
}

type EmailConfig struct {
	ResendAPIKey string `yaml:"resend_api_key"`
	FromAddress  string `yaml:"from_address"`
}

type ServerConfig struct {
	Port           int    `yaml:"port"`
	TelnetPort     int    `yaml:"telnet_port"`
	TelnetTLSPort  int    `yaml:"telnet_tls_port"`
	TelnetTLSCert  string `yaml:"telnet_tls_cert"`
	TelnetTLSKey   string `yaml:"telnet_tls_key"`
	SSHPort        int    `yaml:"ssh_port"`
	FrontendURL    string `yaml:"frontend_url"`
}

type MongoDBConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type GameConfig struct {
	ScriptsDir string `yaml:"scripts_dir"`
	ConfigFile string `yaml:"config_file"`
	StartRoom  int    `yaml:"start_room"`
	BumpRoom   int    `yaml:"bump_room"`
}

var envPattern = regexp.MustCompile(`\$\{([^:}]+)(?::([^}]*))?\}`)

func expandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := envPattern.FindStringSubmatch(match)
		if val, ok := os.LookupEnv(parts[1]); ok {
			return strings.TrimSpace(val)
		}
		if len(parts) > 2 {
			return parts[2]
		}
		return ""
	})
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	expanded := expandEnv(string(data))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}
	// Resolve relative paths based on engine dir
	if !strings.HasPrefix(cfg.Game.ScriptsDir, "/") {
		// Keep as-is, will be resolved relative to working directory
	}
	return &cfg, nil
}
