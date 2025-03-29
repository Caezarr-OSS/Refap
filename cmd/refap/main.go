package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caezarr-oss/refap/config"
	"github.com/caezarr-oss/refap/internal/crawler"
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

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.General.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.General.LogPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
		os.Exit(1)
	}

	// Create crawler instance with configuration
	c := crawler.New(crawler.Config{
		ArtiURL:            cfg.Artifactory.URL,
		BaseDir:            cfg.General.OutputDir,
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
	})

	// Process repository list
	repoListPath := cfg.Artifactory.RepoList
	if !filepath.IsAbs(repoListPath) {
		repoListPath = filepath.Join(cfg.General.OutputDir, repoListPath)
	}

	fmt.Printf("Refap starting...\n")
	fmt.Printf("Artifactory URL: %s\n", cfg.Artifactory.URL)
	fmt.Printf("Repository list: %s\n", repoListPath)
	fmt.Printf("Output directory: %s\n", cfg.General.OutputDir)

	if err := c.ParseRepoList(repoListPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing repository list: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Refap completed successfully")
}
