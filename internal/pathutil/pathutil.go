package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Constants for Windows path handling
const (
	// MaxPathLength is the maximum allowed path length in Windows
	MaxPathLength = 259 // 260 - 1 (null terminator)

	// LongPathPrefix is the prefix for long paths in Windows
	LongPathPrefix = `\\?\`
)

// Reserved filenames in Windows
var reservedNames = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

// Invalid characters in Windows filenames
var invalidCharsWindows = []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}

// IsWindowsOS returns true if the current OS is Windows
func IsWindowsOS() bool {
	return runtime.GOOS == "windows"
}

// SanitizeFilename sanitizes a filename to be compatible with the current OS
// On Windows, it removes invalid characters and checks for reserved names
func SanitizeFilename(filename string) string {
	// Common operations for all platforms
	// Remove leading and trailing whitespace
	filename = strings.TrimSpace(filename)
	
	if IsWindowsOS() {
		// Windows-specific sanitization
		
		// Replace invalid characters
		for _, char := range invalidCharsWindows {
			filename = strings.ReplaceAll(filename, string(char), "_")
		}
		
		// Check for reserved names by comparing the base filename without extension
		base := strings.ToUpper(filepath.Base(filename))
		ext := filepath.Ext(base)
		baseWithoutExt := strings.TrimSuffix(base, ext)
		
		for _, reservedName := range reservedNames {
			if baseWithoutExt == reservedName {
				// Append underscore to avoid reserved name
				return strings.TrimSuffix(filename, ext) + "_" + ext
			}
		}
	}
	
	return filename
}

// SanitizePath sanitizes a full path to be compatible with the current OS
func SanitizePath(path string) string {
	if IsWindowsOS() {
		// Split path into parts
		parts := strings.Split(filepath.ToSlash(path), "/")
		
		// Sanitize each part
		for i, part := range parts {
			if part != "" && i > 0 { // Skip drive letter sanitization
				parts[i] = SanitizeFilename(part)
			}
		}
		
		// Rejoin the path
		result := filepath.FromSlash(strings.Join(parts, "/"))
		
		// Handle long paths
		if len(result) > MaxPathLength && !strings.HasPrefix(result, LongPathPrefix) {
			// Add long path prefix for Windows if needed
			result = LongPathPrefix + result
		}
		
		return result
	}
	
	// For non-Windows systems, just normalize the path
	return filepath.Clean(path)
}

// HandleLongPaths ensures that a path can be accessed even if it exceeds Windows path limits
func HandleLongPaths(path string) string {
	if IsWindowsOS() && len(path) > MaxPathLength && !strings.HasPrefix(path, LongPathPrefix) {
		return LongPathPrefix + path
	}
	return path
}

// SafeJoin safely joins path elements with proper sanitization
func SafeJoin(elements ...string) string {
	// Sanitize each element
	for i, elem := range elements {
		elements[i] = SanitizeFilename(elem)
	}
	
	// Join the path
	result := filepath.Join(elements...)
	
	// Apply additional sanitization for the full path
	return SanitizePath(result)
}

// EnsureDirectoryExists ensures a directory exists, creating it if necessary
func EnsureDirectoryExists(path string) error {
	// Sanitize the path
	sanitizedPath := SanitizePath(path)
	
	// Check if the directory exists
	info, err := os.Stat(sanitizedPath)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return errors.New("path exists but is not a directory")
		}
		return nil
	}
	
	// Create the directory
	return os.MkdirAll(sanitizedPath, 0755)
}

// SafeCreateFile safely creates a file with a sanitized path
func SafeCreateFile(path string) (*os.File, error) {
	// Sanitize the path
	sanitizedPath := SanitizePath(path)
	
	// Ensure parent directory exists
	parent := filepath.Dir(sanitizedPath)
	if err := EnsureDirectoryExists(parent); err != nil {
		return nil, err
	}
	
	// Create the file
	return os.Create(sanitizedPath)
}

// URLToFilePath converts a URL path to a filesystem path
// Handles differences between URL paths (always '/') and local filesystem paths
func URLToFilePath(urlPath string) string {
	// Convert URL path to filesystem path
	// URL paths always use forward slashes, but Windows paths use backslashes
	path := filepath.FromSlash(urlPath)
	
	// Apply sanitization
	return SanitizePath(path)
}

// ConvertURIToFilePath converts a URI to a filesystem path
func ConvertURIToFilePath(uri string) string {
	// Remote "file://" prefix if present
	cleanURI := strings.TrimPrefix(uri, "file://")
	
	// Convert slashes to platform-specific separator
	path := filepath.FromSlash(cleanURI)
	
	// Apply sanitization
	return SanitizePath(path)
}
