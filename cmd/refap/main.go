package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caezarr-oss/refap/config"
	"github.com/caezarr-oss/refap/internal/crawler"
	"github.com/caezarr-oss/refap/internal/pathutil"
)

var (
	// Version information - will be set during build using ldflags
	Version   = "dev"
	CommitSHA = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "refap.toml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Display version information if requested
	if *showVersion {
		fmt.Printf("Refap (Rex Factory Patriot) %s\n", Version)
		fmt.Printf("Commit: %s\n", CommitSHA)
		fmt.Printf("Build Date: %s\n", BuildDate)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Sanitize and ensure output directory exists
	safeOutputDir := pathutil.SanitizePath(cfg.General.OutputDir)
	if err := pathutil.EnsureDirectoryExists(safeOutputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Update the configuration with the sanitized path
	cfg.General.OutputDir = safeOutputDir

	// Sanitize and ensure log directory exists
	safeLogPath := pathutil.SanitizePath(cfg.General.LogPath)
	if err := pathutil.EnsureDirectoryExists(safeLogPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
		os.Exit(1)
	}

	// Update the configuration with the sanitized path
	cfg.General.LogPath = safeLogPath

	// Create crawler instance with configuration
	c := crawler.New(crawler.Config{
		ArtiURL:            cfg.Artifactory.URL,
		BaseDir:            safeOutputDir,
		FileTypes:          cfg.GetFileTypesList(),
		ForceReplace:       cfg.Artifactory.ForceReplace,
		RetryAttempts:      cfg.Download.RetryAttempts,
		Timeout:            cfg.Download.Timeout,
		UseWget:            cfg.Download.UseWget,
		Delay:              cfg.Download.Delay,
		ProxyEnabled:       cfg.Proxy.Enabled,
		ProxyHost:          cfg.Proxy.Host,
		ProxyPort:          cfg.Proxy.Port,
		ProxyUsername:      cfg.Proxy.Username,
		ProxyPassword:      cfg.Proxy.Password,
		AuthType:           cfg.Auth.Type,
		AuthUsername:       cfg.Auth.Username,
		AuthPassword:       cfg.Auth.Password,
		AuthAccessToken:    cfg.Auth.AccessToken,
		FilterMode:         cfg.GetFilterMode(),
		Extensions:         cfg.GetFileTypesList(),
	})

	// Process repository list with safe path handling
	repoListPath := cfg.Artifactory.RepoList
	if !filepath.IsAbs(repoListPath) {
		repoListPath = pathutil.SafeJoin(safeOutputDir, repoListPath)
	} else {
		repoListPath = pathutil.SanitizePath(repoListPath)
	}

	fmt.Printf("Refap starting...\n")
	fmt.Printf("Artifactory URL: %s\n", cfg.Artifactory.URL)
	fmt.Printf("Repository list: %s\n", repoListPath)
	fmt.Printf("Output directory: %s\n", safeOutputDir)
	
	// Print platform-specific information
	if pathutil.IsWindowsOS() {
		fmt.Println("Running on Windows - Using Windows-compatible path handling")
	} else {
		fmt.Println("Running on Unix/Linux - Using Unix path handling")
	}

	if err := c.ParseRepoList(repoListPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing repository list: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Refap completed successfully")
}
