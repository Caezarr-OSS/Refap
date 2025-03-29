package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Default values
const (
	DefaultConcurrentDownloads = 4
	DefaultRetryAttempts       = 3
	DefaultTimeout             = 10
	DefaultDelay               = 1
)

// FileTypesDefault is the default set of file extensions to download
const FileTypesDefault = ".pom,.jar,.war,.xml,.zip,.tar,.tar.gz"

// FilterMode defines the mode of file filtering
type FilterMode string

const (
	// FilterModeNone means no filtering
	FilterModeNone FilterMode = "none"
	// FilterModeWhitelist means only download files with extensions in the whitelist
	FilterModeWhitelist FilterMode = "whitelist"
	// FilterModeBlacklist means download all files except those with extensions in the blacklist
	FilterModeBlacklist FilterMode = "blacklist"
)

// Config represents the application's configuration
type Config struct {
	General struct {
		OutputDir           string `mapstructure:"output_dir"`
		LogPath             string `mapstructure:"log_path"`
		LogLevel            string `mapstructure:"log_level"`
		ConcurrentDownloads int    `mapstructure:"concurrent_downloads"`
	} `mapstructure:"general"`

	Artifactory struct {
		URL          string   `mapstructure:"url"`
		RepoList     string   `mapstructure:"repo_list"`
		Repositories []string `mapstructure:"repositories"`
		FileTypes    string   `mapstructure:"file_types"`
		ForceReplace bool     `mapstructure:"force_replace"`
	} `mapstructure:"artifactory"`

	Files struct {
		FilterMode         string   `mapstructure:"filter_mode"`
		Extensions         []string `mapstructure:"extensions"`
		IncludeMavenMetadata bool   `mapstructure:"include_maven_metadata"`
		CleanHTMLFiles     bool     `mapstructure:"clean_html_files"`
	} `mapstructure:"files"`

	Download DownloadConfig `mapstructure:"download"`
	Proxy    ProxyConfig    `mapstructure:"proxy"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

// DownloadConfig defines download behavior
type DownloadConfig struct {
	RetryAttempts int  `mapstructure:"retry_attempts"`
	Timeout       int  `mapstructure:"timeout"`
	UseWget       bool `mapstructure:"use_wget"`
	Delay         int  `mapstructure:"delay"`
}

// ProxyConfig defines proxy configuration
type ProxyConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// AuthConfig defines the authentication configuration
type AuthConfig struct {
	Type        string `mapstructure:"type"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	AccessToken string `mapstructure:"access_token"`
}

// GetValidAuthTypes returns the list of supported authentication types
func GetValidAuthTypes() []string {
	return []string{"none", "basic", "token"}
}

// IsValidAuthType checks if the authentication type is valid
func IsValidAuthType(authType string) bool {
	for _, validType := range GetValidAuthTypes() {
		if authType == validType {
			return true
		}
	}
	return false
}

// IsValidFilterMode checks if the filter mode is valid
func IsValidFilterMode(mode string) bool {
	return mode == string(FilterModeNone) || 
	       mode == string(FilterModeWhitelist) || 
		   mode == string(FilterModeBlacklist)
}

// GetFilterMode returns the filter mode as a FilterMode type
func (c *Config) GetFilterMode() FilterMode {
	mode := FilterMode(c.Files.FilterMode)
	if !IsValidFilterMode(string(mode)) {
		return FilterModeNone
	}
	return mode
}

// GetFileTypesList returns the list of file types to download as a slice
func (c *Config) GetFileTypesList() []string {
	if c.Artifactory.FileTypes == "" {
		c.Artifactory.FileTypes = FileTypesDefault
	}
	return strings.Split(c.Artifactory.FileTypes, ",")
}

// GetRepositoryList returns the list of repositories to download
// It prioritizes the repositories defined in the TOML config over the repo_list file
func (c *Config) GetRepositoryList() ([]string, error) {
	// If repositories are specified in the TOML file, use them
	if len(c.Artifactory.Repositories) > 0 {
		return c.Artifactory.Repositories, nil
	}

	// Otherwise try to load from repo_list file
	if c.Artifactory.RepoList == "" {
		return nil, errors.New("no repositories configured: neither 'repositories' nor 'repo_list' is set")
	}

	// Load from file
	repoListPath := c.Artifactory.RepoList
	if !filepath.IsAbs(repoListPath) {
		repoListPath = filepath.Join(c.General.OutputDir, repoListPath)
	}

	file, err := os.Open(repoListPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo list file: %w", err)
	}
	defer file.Close()

	var repos []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		repo := strings.TrimSpace(strings.Replace(strings.Replace(scanner.Text(), "\n", "", -1), "\r", "", -1))
		if repo != "" {
			repos = append(repos, repo)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning repo list file: %w", err)
	}

	if len(repos) == 0 {
		return nil, errors.New("no repositories found in repo list file")
	}

	return repos, nil
}

// ShouldIncludeFile checks if a file should be included based on the filter settings
func (c *Config) ShouldIncludeFile(filename string) bool {
	// Special case for maven-metadata.xml if configured to include it
	if c.Files.IncludeMavenMetadata && strings.HasSuffix(filename, "maven-metadata.xml") {
		return true
	}

	// Get the file extension
	ext := filepath.Ext(filename)
	if ext == "" {
		// Check for .tar.gz which would be missed by filepath.Ext
		if strings.HasSuffix(filename, ".tar.gz") {
			ext = ".tar.gz"
		} else {
			// No extension, apply default rules
			mode := c.GetFilterMode()
			return mode != FilterModeWhitelist // Include if not in whitelist mode
		}
	}

	// Check the specified filter mode
	mode := c.GetFilterMode()

	switch mode {
	case FilterModeNone:
		// Default extensions from the artifactory section
		for _, defaultExt := range c.GetFileTypesList() {
			if strings.HasSuffix(filename, defaultExt) {
				return true
			}
		}
		return false

	case FilterModeWhitelist:
		// Only include files with extensions in the whitelist
		for _, allowedExt := range c.Files.Extensions {
			if ext == allowedExt || strings.HasSuffix(filename, allowedExt) {
				return true
			}
		}
		return false

	case FilterModeBlacklist:
		// Include all files except those with extensions in the blacklist
		for _, blockedExt := range c.Files.Extensions {
			if ext == blockedExt || strings.HasSuffix(filename, blockedExt) {
				return false
			}
		}
		return true

	default:
		// Should not happen
		return false
	}
}

// LoadConfig loads the TOML configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	setDefaults()

	if configPath == "" {
		defaultPath, _ := filepath.Abs("./refap.toml")
		configPath = defaultPath
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found at path: %s", configPath)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults() {
	viper.SetDefault("general.output_dir", "./downloads")
	viper.SetDefault("general.log_path", "./logs")
	viper.SetDefault("general.log_level", "info")
	viper.SetDefault("general.concurrent_downloads", DefaultConcurrentDownloads)

	viper.SetDefault("artifactory.url", "http://10.29.204.181:8082/artifactory/list/")
	viper.SetDefault("artifactory.repo_list", "liste_arti.csv")
	viper.SetDefault("artifactory.file_types", FileTypesDefault)
	viper.SetDefault("artifactory.force_replace", false)

	viper.SetDefault("files.filter_mode", "none")
	viper.SetDefault("files.include_maven_metadata", true)
	viper.SetDefault("files.clean_html_files", true)

	viper.SetDefault("download.retry_attempts", DefaultRetryAttempts)
	viper.SetDefault("download.timeout", DefaultTimeout)
	viper.SetDefault("download.use_wget", true)
	viper.SetDefault("download.delay", DefaultDelay)

	viper.SetDefault("proxy.enabled", false)

	viper.SetDefault("auth.type", "none")
}

// validateConfig validates the configuration for coherence
func validateConfig(cfg *Config) error {
	// Validate artifactory URL
	if cfg.Artifactory.URL == "" {
		return errors.New("artifactory URL cannot be empty")
	}

	// Check either repositories in TOML or repo_list is provided
	if len(cfg.Artifactory.Repositories) == 0 && cfg.Artifactory.RepoList == "" {
		return errors.New("either 'repositories' or 'repo_list' must be specified in the configuration")
	}

	// Validate filter mode
	if !IsValidFilterMode(cfg.Files.FilterMode) {
		return fmt.Errorf("invalid filter mode '%s', must be one of: none, whitelist, blacklist", cfg.Files.FilterMode)
	}

	// Validate download configuration
	if cfg.Download.RetryAttempts < 0 {
		return errors.New("retry attempts cannot be negative")
	}

	if cfg.Download.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if cfg.Download.Delay < 0 {
		return errors.New("delay cannot be negative")
	}

	// Validate proxy configuration
	if cfg.Proxy.Enabled {
		if cfg.Proxy.Host == "" {
			return errors.New("proxy host cannot be empty when proxy is enabled")
		}
		if cfg.Proxy.Port <= 0 || cfg.Proxy.Port > 65535 {
			return errors.New("proxy port must be between 1 and 65535")
		}
	}

	// Validate authentication
	return validateAuthConfig(&cfg.Auth)
}

// validateAuthConfig validates the authentication configuration
func validateAuthConfig(auth *AuthConfig) error {
	if !IsValidAuthType(auth.Type) {
		return fmt.Errorf("auth type %s is not supported, valid values are: %v", auth.Type, GetValidAuthTypes())
	}

	if auth.Type == "basic" {
		if auth.Username == "" {
			return errors.New("username cannot be empty for basic authentication")
		}
		if auth.Password == "" {
			return errors.New("password cannot be empty for basic authentication")
		}
	}

	if auth.Type == "token" && auth.AccessToken == "" {
		return errors.New("access token cannot be empty for token authentication")
	}

	return nil
}
