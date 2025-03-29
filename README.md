# Refap (Rex Factory Patriot)

<img src="assets/img/refap.png" alt="Refap Logo" width="200"/>

Refap - **R**ex **F**actory **P**atriot (an anagram of "Export Artifactory") is a tool designed to crawl Artifactory repositories via HTML pages and download specified files.

## Description

Refap allows you to navigate through Artifactory index pages and automatically download specified files (JAR, POM, WAR, ZIP, etc.) while preserving the directory structure. Originally written in Python (wget_arti.py), it has been converted to Go for better performance and more features.

## Features

- Crawl and download files from Artifactory repositories
- Filter files based on extensions (whitelist or blacklist)
- Special handling for maven-metadata.xml files
- Authentication support (Basic Auth and Bearer Token)
- Proxy support
- Parallel downloads
- Configurable retry mechanism
- HTML cleanup after processing

## Installation

```bash
# Clone the repository
git clone https://github.com/caezarr-oss/refap.git
cd refap

# Build the executable
go build -o refap ./cmd/refap
```

## Development

Refap uses [Task](https://taskfile.dev) for development operations.

```bash
# Build for your current platform
task build

# Run tests
task test

# Build for all platforms
task release
```

## Usage

### Basic Usage

Refap automatically looks for a configuration file named `refap.toml` in the same directory as the executable. If found, it will use that configuration without requiring any additional parameters.

```bash
# Run with default configuration (looks for refap.toml in current directory)
./refap

# Specify a custom configuration file
./refap -c custom-config.toml
./refap --config custom-config.toml

# Show version
./refap -v
./refap --version
```

## Configuration Guide

Refap uses a TOML configuration file to control all aspects of its behavior. Below is a detailed explanation of all available configuration options.

### Configuration Overview

The configuration file is divided into several sections:

1. **general**: Basic application settings
2. **artifactory**: Artifactory connection and repository settings
3. **files**: File filtering options
4. **download**: Download behavior settings
5. **proxy**: Proxy server configuration
6. **auth**: Authentication settings

### General Settings

```toml
[general]
output_dir = "./downloads"
log_path = "./logs/refap.log"
log_level = "info"
concurrent_downloads = 4
```

- **output_dir**: Directory where downloaded files will be stored
- **log_path**: Path to the log file
- **log_level**: Log verbosity (debug, info, warn, error)
- **concurrent_downloads**: Maximum number of parallel downloads

### Artifactory Settings

```toml
[artifactory]
url = "http://artifactory.example.com:8082/artifactory/list/"
repositories = [
  "maven-central/org/springframework/spring-core",
  "maven-central/com/google/guava/guava",
  "maven-central/org/apache/commons/commons-lang3"
]
repo_list = "liste_arti.csv"
file_types = ".pom,.jar,.war,.xml,.zip,.tar,.tar.gz"
force_replace = false
```

- **url**: Base URL of the Artifactory server
- **repositories**: List of specific repositories to export (array format)
- **repo_list**: Path to a CSV file containing additional repositories (one per line)
- **file_types**: Legacy setting for file types to download if filter_mode is "none"
- **force_replace**: Whether to overwrite existing files during download

### File Filtering Settings

```toml
[files]
filter_mode = "whitelist"
extensions = [".jar", ".pom", ".war", ".zip", ".tar", ".tar.gz"]
include_maven_metadata = true
clean_html_files = true
```

- **filter_mode**: Controls how files are filtered during the download process:
  - `whitelist`: Only download files with extensions listed in the `extensions` array. This is the most restrictive mode and ensures only specific file types are downloaded.
  - `blacklist`: Download all files EXCEPT those with extensions listed in the `extensions` array. This mode is useful when you want to exclude specific file types.
  - `none`: Ignore the `extensions` setting and use the legacy `file_types` setting from the `[artifactory]` section instead. This mode exists for backward compatibility with older configurations.

- **extensions**: List of file extensions to include (in whitelist mode) or exclude (in blacklist mode). Extensions should include the leading dot (e.g., ".jar").

- **include_maven_metadata**: When set to `true`, always include maven-metadata.xml files regardless of the filter settings. This is useful because these files contain important metadata about Maven artifacts but might not match your extension filters.

- **clean_html_files**: When set to `true`, delete temporary HTML index files after processing is complete. These are the intermediate files downloaded during crawling and are not usually needed after the export is finished.

### Download Settings

```toml
[download]
retry_attempts = 3
timeout = 10
delay = 1
```

- **retry_attempts**: Number of download retries for failed requests
- **timeout**: HTTP request timeout in seconds
- **delay**: Delay between retry attempts in seconds

### Proxy Settings

```toml
[proxy]
enabled = false
host = ""
port = 0
username = ""
password = ""
```

- **enabled**: Whether to use a proxy server
- **host**: Proxy server hostname or IP address
- **port**: Proxy server port number
- **username**: Username for proxy authentication (if required)
- **password**: Password for proxy authentication (if required)

### Authentication Settings

```toml
[auth]
type = "none"
username = ""
password = ""
access_token = ""
```

- **type**: Authentication type:
  - `none`: No authentication
  - `basic`: HTTP Basic authentication (requires username and password)
  - `token`: Token-based authentication (requires access_token)
- **username**: Username for Basic authentication
- **password**: Password for Basic authentication
- **access_token**: Access token for token-based authentication

## License

See the LICENSE file for details.