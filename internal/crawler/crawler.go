package crawler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caezarr-oss/refap/config"
	"github.com/caezarr-oss/refap/internal/pathutil"
)

// Configuration options for the crawler
type Config struct {
	ArtiURL         string
	BaseDir         string
	FileTypes       []string
	ForceReplace    bool
	RetryAttempts   int
	Timeout         int
	UseWget         bool
	Delay           int
	ProxyEnabled    bool
	ProxyHost       string
	ProxyPort       int
	ProxyUsername   string
	ProxyPassword   string
	AuthType        string
	AuthUsername    string
	AuthPassword    string
	AuthAccessToken string
	FilterMode      config.FilterMode
	Extensions      []string
	IncludeMavenMetadata bool
	CleanHTMLFiles  bool
}

// New creates a new Crawler with the provided configuration
func New(config Config) *Crawler {
	return &Crawler{
		config: config,
		htmlFiles: make([]string, 0),
	}
}

// Crawler handles the artifactory crawling operations
type Crawler struct {
	config Config
	htmlFiles []string // List of all HTML index files created
}

// ParseIndex parses an HTML index file and downloads all referenced files
func (c *Crawler) ParseIndex(file, path, artiURL string) error {
	// Sanitize file and path for Windows compatibility
	safeFile := pathutil.SanitizePath(file)
	safePath := pathutil.SanitizePath(path)
	
	// Add the HTML file to the list for potential cleanup later
	absPath, err := filepath.Abs(safeFile)
	if err == nil {
		c.htmlFiles = append(c.htmlFiles, absPath)
	}

	// Detect encoding and read file
	f, err := os.Open(safeFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Change to the specified directory
	if err := os.Chdir(safePath); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", safePath, err)
	}

	// Read the file line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimLeft(strings.Replace(scanner.Text(), "\t", "", -1), " ")

		// Check if the line starts with "<a href=" or "<pre><a href="
		if strings.HasPrefix(line, "<a href=") || strings.HasPrefix(line, "<pre><a href=") {
			// Extract href value
			hrefStartIndex := strings.Index(line, "href=") + len("href=")
			if hrefStartIndex < len("href=") {
				continue
			}
			
			hrefEndIndex := strings.Index(line[hrefStartIndex:], "\"")
			if hrefEndIndex < 0 {
				continue
			}
			hrefEndIndex += hrefStartIndex
			urlValue := line[hrefStartIndex+1:hrefEndIndex]

			// Extract element value (text between <a> tags)
			elStartIndex := strings.Index(line, ">") + 1
			if elStartIndex < 1 {
				continue
			}
			
			elEndIndex := strings.Index(line[elStartIndex:], "</a>")
			if elEndIndex < 0 {
				continue
			}
			elEndIndex += elStartIndex
			elValue := line[elStartIndex:elEndIndex]

			// Check if it's a file we want to download
			isTargetFile := c.shouldDownloadFile(urlValue)

			if isTargetFile {
				if !strings.HasSuffix(elValue, "/") {
					// Check if file already exists and if we should skip it
					if !c.config.ForceReplace {
						safeElPath := pathutil.SafeJoin(safePath, elValue)
						if _, err := os.Stat(safeElPath); err == nil {
							continue
						}
					}

					fmt.Printf("Downloading %s in %s\n", elValue, safePath)
					if err := c.downloadFile(elValue, artiURL+urlValue); err != nil {
						// Log failed download and continue
						// Use HOME directory instead of hard-coded USERPROFILE for cross-platform compatibility
						logDir := os.Getenv("HOME")
						if pathutil.IsWindowsOS() {
							logDir = os.Getenv("USERPROFILE")
						}
						failLogPath := pathutil.SafeJoin(logDir, "Documents", "EXPORT_ARTI", "failed_download.txt")
						if err := pathutil.EnsureDirectoryExists(filepath.Dir(failLogPath)); err == nil {
							failLog, err := os.OpenFile(failLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
							if err == nil {
								fmt.Fprintf(failLog, "wget --timeout=%d --tries=%d -O %s %s\n", c.config.Timeout, c.config.RetryAttempts, elValue, artiURL+urlValue)
								failLog.Close()
							}
						}
					}
					// Wait between downloads as specified in config
					time.Sleep(time.Duration(c.config.Delay) * time.Second)
				} else if strings.Contains(elValue, "..") {
					continue
				}
			} else {
				// This is a directory, crawl recursively
				if strings.Contains(elValue, "..") {
					continue
				}

				// Create directory with safe path handling
				dirPath := pathutil.SafeJoin(safePath, elValue)
				if err := pathutil.EnsureDirectoryExists(dirPath); err != nil {
					fmt.Printf("Failed to create directory %s: %v\n", dirPath, err)
					continue
				}

				// Change to new directory
				if err := os.Chdir(dirPath); err != nil {
					fmt.Printf("Failed to change to directory %s: %v\n", dirPath, err)
					continue
				}

				// Generate index file name - sanitize it for Windows
				indexName := pathutil.SanitizeFilename(strings.Replace(elValue, "/", "", -1) + "-index.html")
				
				// Download index file
				fmt.Printf("Downloading index for %s\n", elValue)
				if err := c.downloadFile(indexName, artiURL+urlValue); err != nil {
					fmt.Printf("Failed to download index %s: %v\n", indexName, err)
					continue
				}

				// Parse the new index file
				fmt.Printf("Parsing: %s / %s in new path: %s\n", indexName, artiURL+urlValue, dirPath)
				if err := c.ParseIndex(indexName, filepath.Join(dirPath, "/"), artiURL+urlValue); err != nil {
					fmt.Printf("Failed to parse index %s: %v\n", indexName, err)
				}
				
				fmt.Printf("File %s parsed\n", file)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning file: %w", err)
	}

	return nil
}

// shouldDownloadFile checks if a file should be downloaded based on filter settings
func (c *Crawler) shouldDownloadFile(filePath string) bool {
	// Special case for maven-metadata.xml if configured to include it
	if c.config.IncludeMavenMetadata && strings.HasSuffix(filePath, "maven-metadata.xml") {
		return true
	}

	// Get the file extension
	ext := filepath.Ext(filePath)
	if ext == "" {
		// Check for .tar.gz which would be missed by filepath.Ext
		if strings.HasSuffix(filePath, ".tar.gz") {
			ext = ".tar.gz"
		} else {
			// No extension, apply default rules
			return c.config.FilterMode != config.FilterModeWhitelist // Include if not in whitelist mode
		}
	}

	// Apply filter based on mode
	switch c.config.FilterMode {
	case config.FilterModeNone:
		// Use the legacy file types list
		for _, defaultExt := range c.config.FileTypes {
			if strings.HasSuffix(filePath, defaultExt) {
				return true
			}
		}
		return false

	case config.FilterModeWhitelist:
		// Only include files with extensions in the whitelist
		for _, allowedExt := range c.config.Extensions {
			if ext == allowedExt || strings.HasSuffix(filePath, allowedExt) {
				return true
			}
		}
		return false

	case config.FilterModeBlacklist:
		// Include all files except those with extensions in the blacklist
		for _, blockedExt := range c.config.Extensions {
			if ext == blockedExt || strings.HasSuffix(filePath, blockedExt) {
				return false
			}
		}
		return true

	default:
		// Should not happen
		return false
	}
}

// downloadFile downloads a file from the given URL and saves it to the specified path
func (c *Crawler) downloadFile(filepath, urlStr string) error {
	// Sanitize the filepath for Windows compatibility
	safeFilepath := pathutil.SanitizeFilename(filepath)
	
	// Configure client with timeout
	client := &http.Client{
		Timeout: time.Duration(c.config.Timeout) * time.Second,
	}

	// Configure proxy if enabled
	if c.config.ProxyEnabled && c.config.ProxyHost != "" && c.config.ProxyPort > 0 {
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", c.config.ProxyHost, c.config.ProxyPort),
		}

		if c.config.ProxyUsername != "" && c.config.ProxyPassword != "" {
			proxyURL.User = url.UserPassword(c.config.ProxyUsername, c.config.ProxyPassword)
		}

		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	// Create request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}

	// Add authentication if configured
	switch c.config.AuthType {
	case "basic":
		if c.config.AuthUsername != "" && c.config.AuthPassword != "" {
			req.SetBasicAuth(c.config.AuthUsername, c.config.AuthPassword)
		}
	case "token":
		if c.config.AuthAccessToken != "" {
			req.Header.Add("Authorization", "Bearer "+c.config.AuthAccessToken)
		}
	}

	// Add a user agent to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// Perform request with retry logic
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < c.config.RetryAttempts; attempt++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("failed to download %s: status code %d", urlStr, resp.StatusCode)
			resp.Body.Close()
		}

		// Wait before retrying
		if attempt < c.config.RetryAttempts-1 {
			time.Sleep(time.Duration(c.config.Delay) * time.Second)
		}
	}

	if lastErr != nil {
		return lastErr
	}

	defer resp.Body.Close()

	// Create file with safe path handling
	outFile, err := pathutil.SafeCreateFile(safeFilepath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}

// CleanupHTMLFiles removes all HTML index files created during the crawling process
func (c *Crawler) CleanupHTMLFiles() error {
	var lastErr error
	for _, file := range c.htmlFiles {
		safeFile := pathutil.SanitizePath(file)
		if err := os.Remove(safeFile); err != nil {
			lastErr = err
			fmt.Printf("Failed to remove HTML file %s: %v\n", safeFile, err)
		}
	}
	return lastErr
}

// ProcessRepositories processes all repositories defined in the configuration
func (c *Crawler) ProcessRepositories(repoList []string) error {
	// Ensure the base directory exists and is sanitized
	safeBaseDir := pathutil.SanitizePath(c.config.BaseDir)
	if err := pathutil.EnsureDirectoryExists(safeBaseDir); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", safeBaseDir, err)
	}

	// Create the export directory if it doesn't exist
	exportDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents", "EXPORT_ARTI")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// Process each repository in the list
	for _, repo := range repoList {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}

		// Create repo-specific directory if it doesn't exist
		repoDir := filepath.Join(safeBaseDir, strings.ReplaceAll(repo, "/", string(os.PathSeparator)))
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			fmt.Printf("Failed to create repository directory: %v\n", err)
			continue
		}

		// Create index filename
		mainIndexName := strings.Replace(repo, "/", "_", -1) + "-index.html"
		
		// Download main index file
		fmt.Printf("Downloading main index for repo: %s\n", repo)
		if err := c.downloadFile(mainIndexName, c.config.ArtiURL+repo); err != nil {
			fmt.Printf("Failed to download index for repo %s: %v\n", repo, err)
			continue
		}

		// Parse the main index
		fmt.Printf("Parsing index: %s\n", mainIndexName)
		if err := c.ParseIndex(mainIndexName, safeBaseDir+"/", c.config.ArtiURL+repo); err != nil {
			fmt.Printf("Failed to parse index for repo %s: %v\n", repo, err)
		}
	}

	// Clean up HTML files if configured to do so
	if err := c.CleanupHTMLFiles(); err != nil {
		fmt.Printf("Warning: Error during HTML cleanup: %v\n", err)
	}

	return nil
}

// ParseRepoList reads a list of repositories from a file and processes each one
func (c *Crawler) ParseRepoList(repoListFile string) error {
	// Open the repository list file
	file, err := os.Open(repoListFile)
	if err != nil {
		return fmt.Errorf("failed to open repo list file: %w", err)
	}
	defer file.Close()

	// Process each repository in the list
	var repos []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		repo := strings.TrimSpace(strings.Replace(strings.Replace(scanner.Text(), "\n", "", -1), "\r", "", -1))
		if repo != "" {
			repos = append(repos, repo)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning repo list file: %w", err)
	}

	return c.ProcessRepositories(repos)
}
