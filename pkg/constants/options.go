package constants

import "time"

const (
	ConfigFileName = "config.yaml"
)

// Default values for jxscout options
const (
	// Server configuration
	DefaultPort     = 3333
	DefaultHostname = "localhost"
	DefaultProxyURL = ""

	// Jxscout configuration
	DefaultProjectName = "default"
	DefaultDebug       = false

	// Concurrency configuration
	DefaultAssetFetchConcurrency      = 5
	DefaultAssetSaveConcurrency       = 5
	DefaultBeautifierConcurrency      = 5
	DefaultChunkDiscovererConcurrency = 5
	DefaultASTAnalyzerConcurrency     = 5

	// Chunk discovery configuration
	DefaultChunkDiscovererBruteForceLimit = 3000

	DefaultJavascriptRequestsCacheTTL = 10 * time.Minute
	DefaultHTMLRequestsCacheTTL       = 10 * time.Minute

	// Rate limiting configuration
	DefaultRateLimitingMaxRequestsPerSecond = 2
	DefaultRateLimitingMaxRequestsPerMinute = 0

	// JS ingestion configuration
	DefaultDownloadReferedJS = false

	// Logging configuration
	DefaultLogBufferSize    = 10000
	DefaultLogFileMaxSizeMB = 10

	// Overrides configuration
	DefaultCaidoHostname                = "localhost"
	DefaultCaidoPort                    = 8080
	DefaultOverrideContentCheckInterval = time.Second * 5
)

// Flag names for jxscout options
const (
	// Server configuration
	FlagPort     = "port"
	FlagHostname = "hostname"
	FlagProxyURL = "proxy-url"

	// Jxscout configuration
	FlagProjectName = "project-name"
	FlagScope       = "scope"
	FlagDebug       = "debug"

	// Concurrency configuration
	FlagAssetFetchConcurrency      = "fetch-concurrency"
	FlagAssetSaveConcurrency       = "save-concurrency"
	FlagBeautifierConcurrency      = "beautifier-concurrency"
	FlagChunkDiscovererConcurrency = "chunk-discoverer-concurrency"
	FlagASTAnalyzerConcurrency     = "ast-analyzer-concurrency"

	// Chunk discovery configuration
	FlagChunkDiscovererBruteForceLimit = "chunk-discoverer-bruteforce-limit"

	// Cache configuration
	FlagJavascriptRequestsCacheTTL = "js-requests-cache-ttl"
	FlagHTMLRequestsCacheTTL       = "html-requests-cache-ttl"

	// Rate limiting configuration
	FlagRateLimitingMaxRequestsPerSecond = "rate-limiter-max-requests-per-second"
	FlagRateLimitingMaxRequestsPerMinute = "rate-limiter-max-requests-per-minute"

	// JS ingestion configuration
	FlagDownloadReferedJS = "download-refered-js"

	// Logging configuration
	FlagLogBufferSize    = "log-buffer-size"
	FlagLogFileMaxSizeMB = "log-file-max-size-mb"

	// Overrides configuration
	FlagCaidoHostname                = "caido-hostname"
	FlagCaidoPort                    = "caido-port"
	FlagOverrideContentCheckInterval = "override-content-check-interval"
)

// Descriptions for jxscout options
const (
	// Server configuration
	DescriptionHostname = "the hostname where jxscout will listen for requests"
	DescriptionPort     = "the port where jxscout will listen for requests"
	DescriptionProxyURL = "proxy URL to use for HTTP requests (e.g. http://proxy:8080, socks5://proxy:1080)"

	// Jxscout configuration
	DescriptionProjectName = "name of your project folder where downloaded files will be stored"
	DescriptionScope       = "comma-separated list of patterns to filter requests (e.g. *google*,*youtube*)"
	DescriptionDebug       = "turn on detailed logs for troubleshooting"

	// Concurrency configuration
	DescriptionAssetFetchConcurrency      = "how many files to download at once (for chunks and source maps)"
	DescriptionAssetSaveConcurrency       = "how many files to save to disk at once"
	DescriptionBeautifierConcurrency      = "how many files to beautify at once"
	DescriptionChunkDiscovererConcurrency = "how many chunk discovery processes to run at once"
	DescriptionASTAnalyzerConcurrency     = "how many AST analysis processes to run at once"

	// Chunk discovery configuration
	DescriptionChunkDiscovererBruteForceLimit = "how many potential chunks to bruteforce when automatic discovery fails"

	// Cache configuration
	DescriptionJavascriptRequestsCacheTTL = "how long to wait before re-downloading the same JS file"
	DescriptionHTMLRequestsCacheTTL       = "how long to wait before re-downloading the same HTML page"

	// Rate limiting configuration
	DescriptionRateLimitingMaxRequestsPerMinute = "max requests per minute for source maps and chunk discovery (0 = unlimited)"
	DescriptionRateLimitingMaxRequestsPerSecond = "max requests per second for source maps and chunk discovery (0 = unlimited)"

	// JS ingestion configuration
	DescriptionDownloadReferedJS = "download JS files from out-of-scope domains if they're linked from in-scope pages"

	// Logging configuration
	DescriptionLogBufferSize    = "how many log lines to show in the logs panel"
	DescriptionLogFileMaxSizeMB = "max size of the log file in MB"

	// Overrides configuration
	DescriptionCaidoHostname                = "hostname where Caido is running"
	DescriptionCaidoPort                    = "port where Caido is running"
	DescriptionOverrideContentCheckInterval = "interval at which to check for changes in override content and update match/replace rules"
)
