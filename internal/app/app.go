package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bethropolis/dir-dumper/internal/config"
	"github.com/bethropolis/dir-dumper/internal/ignore"
	"github.com/bethropolis/dir-dumper/internal/logger"
	"github.com/bethropolis/dir-dumper/internal/printer"
	"github.com/bethropolis/dir-dumper/internal/walker"
	"github.com/fatih/color"
)

// App encapsulates the main application functionality
type App struct {
	cfg    *config.Config
	log    *logger.Logger
	Output io.Writer // Changed from output to Output (exported)
}

// New creates a new App instance
func New(cfg *config.Config) *App {
	// Configure color globally
	color.NoColor = !cfg.UseColors

	// Set up output destination
	var output io.Writer = os.Stdout
	if cfg.OutputFile != "" {
		file, err := os.Create(cfg.OutputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create output file: %v\n", err)
			os.Exit(1)
		}
		// Note: file will be closed by main function
		output = file
	}

	// Set up logger
	log := logger.New(os.Stderr, cfg.Verbose, cfg.UseColors)

	// Apply log level if specified (overrides verbose/quiet flags)
	if cfg.LogLevel != "" {
		log.SetLevel(cfg.LogLevel)
	} else if cfg.Quiet {
		// For backward compatibility
		log.WithLevel(logger.LevelWarn)
	}

	return &App{
		cfg:    cfg,
		log:    log,
		Output: output,
	}
}

// Run executes the main application logic
func (a *App) Run() {
	startTime := time.Now() // Start timer for overall execution

	// Show version and exit if requested
	if a.cfg.ShowVersion {
		fmt.Printf("dir-dumper version %s\n", a.cfg.Version)
		os.Exit(0)
	}

	// Handle timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc

	if a.cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), a.cfg.Timeout)
		defer cancel()

		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Fprintf(os.Stderr, "\nTimeout of %v reached. Exiting.\n", a.cfg.Timeout)
				os.Exit(1)
			}
		}()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	// Helper for info messages, suppressed by quiet flag
	infoLog := func(format string, args ...interface{}) {
		if !a.cfg.Quiet {
			a.log.Info(format, args...)
		}
	}

	if a.log.VerboseMode {
		a.log.Debug("Verbose mode enabled")
		a.log.Debug("Color output: %v", a.cfg.UseColors)
		a.log.Debug("Directory: %s", a.cfg.RootDir)
		a.log.Debug("Concurrent mode: %v (workers: %d)", a.cfg.Concurrent, a.cfg.MaxWorkers)
		a.log.Debug("Max file size: %d MB", a.cfg.MaxFileSizeMB)
		a.log.Debug("Ignore settings: hidden=%v, git=%v",
			a.cfg.IgnoreHidden, a.cfg.IgnoreGit)
		if a.cfg.CustomIgnore != "" {
			a.log.Debug("Custom ignore patterns: %s", a.cfg.CustomIgnore)
		}
		if a.cfg.Extensions != "" {
			a.log.Debug("Extensions filter: %s", a.cfg.Extensions)
		}
	}

	// --- Directory validation ---
	absRootDir, err := filepath.Abs(a.cfg.RootDir)
	if err != nil {
		a.log.Error("Invalid root directory path '%s': %v", a.cfg.RootDir, err)
		os.Exit(1)
	}

	// Check if directory exists
	dirInfo, err := os.Stat(absRootDir)
	if err != nil {
		if os.IsNotExist(err) {
			a.log.Error("Root directory '%s' not found.", absRootDir)
		} else {
			a.log.Error("Could not access root directory '%s': %v", absRootDir, err)
		}
		os.Exit(1)
	}
	if !dirInfo.IsDir() {
		a.log.Error("Specified path '%s' is not a directory.", absRootDir)
		os.Exit(1)
	}

	// --- Parse custom ignore patterns ---
	var customPatterns []string
	if a.cfg.CustomIgnore != "" {
		customPatterns = strings.Split(a.cfg.CustomIgnore, ",")
		for i, pattern := range customPatterns {
			customPatterns[i] = strings.TrimSpace(pattern) // Trim whitespace
		}
		infoLog("Using custom ignore patterns: %v", customPatterns)
	}

	// --- Parse file extensions ---
	var fileExtensions map[string]struct{}
	if a.cfg.Extensions != "" {
		fileExtensions = make(map[string]struct{})
		extList := strings.Split(a.cfg.Extensions, ",")
		var addedExts []string
		for _, ext := range extList {
			cleanExt := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(ext, ".")))
			if cleanExt != "" {
				fileExtensions[cleanExt] = struct{}{}
				addedExts = append(addedExts, "."+cleanExt)
			}
		}
		infoLog("Filtering enabled. Only including extensions: %s", strings.Join(addedExts, ", "))
	} else {
		infoLog("No extension filtering (including all file types).")
	}

	// Print effective settings
	if a.cfg.IgnoreHidden {
		infoLog("Ignoring hidden files/directories (starting with '.').")
	} else {
		infoLog("Including hidden files/directories.")
	}

	// --- Initialize ignore matcher ---
	ignoreOptions := []ignore.Option{
		ignore.WithLogger(a.log),
		ignore.WithHiddenIgnore(a.cfg.IgnoreHidden),
		ignore.WithGitIgnore(a.cfg.IgnoreGit),
	}
	if len(customPatterns) > 0 {
		ignoreOptions = append(ignoreOptions, ignore.WithCustomRules(customPatterns))
	}

	matcher, err := ignore.New(absRootDir, ignoreOptions...)
	if err != nil {
		a.log.Error("Error initializing ignore rules: %v", err)
		os.Exit(1)
	}

	// --- Create the printer ---
	p := printer.New()
	p.WithOutput(a.Output)
	p.WithColors(a.cfg.UseColors)

	// Enable JSON output if requested
	if a.cfg.JSONOutput {
		a.log.Debug("JSON output mode enabled")
		p.WithJSON(true)
		// Disable colors in JSON mode regardless of other settings
		p.WithColors(false)
	} else if a.cfg.MarkdownOutput {
		a.log.Debug("Markdown output mode enabled")
		p.WithMarkdown(true)
		// Disable colors in Markdown mode regardless of other settings
		p.WithColors(false)
	}

	// --- Set up walk options ---
	var walkOptions []walker.Option

	// Add the options correctly
	walkOptions = append(walkOptions,
		walker.WithLogger(a.log),
		walker.WithConcurrency(a.cfg.Concurrent),
		walker.WithMaxWorkers(a.cfg.MaxWorkers),
	)

	// Add progress option if enabled
	if a.cfg.ShowProgress {
		a.log.Debug("Progress display enabled")

		// Create a progress handler
		walkOptions = append(walkOptions, walker.WithProgress(func(stats walker.ProgressStats) {
			// Only print to stderr to avoid interfering with regular output
			if !a.cfg.Quiet {
				var statusLine string

				if stats.CurrentFilePath != "" {
					// Truncate the path if it's too long
					path := stats.CurrentFilePath
					if len(path) > 40 {
						path = "..." + path[len(path)-37:]
					}

					statusLine = fmt.Sprintf("\rProcessing: %-40s | Files: %d/%d | Dirs: %d",
						path,
						stats.ProcessedFiles,
						stats.TotalFiles,
						stats.TotalDirs)
				} else {
					statusLine = fmt.Sprintf("\rScanning... | Files: %d/%d | Dirs: %d",
						stats.ProcessedFiles,
						stats.TotalFiles,
						stats.TotalDirs)
				}

				// Print with carriage return to overwrite previous line
				fmt.Fprint(os.Stderr, statusLine)
			}
		}))
	}

	// Add extension filtering if specified
	if len(fileExtensions) > 0 {
		var extList []string
		for ext := range fileExtensions {
			extList = append(extList, ext)
		}
		walkOptions = append(walkOptions, walker.WithExtensions(extList))
	}

	// Convert MB to bytes for MaxFileSize if specified
	if a.cfg.MaxFileSizeMB > 0 {
		maxSizeBytes := a.cfg.MaxFileSizeMB * 1024 * 1024
		walkOptions = append(walkOptions, walker.WithMaxFileSize(maxSizeBytes))
		infoLog("Ignoring files larger than %d MB.", a.cfg.MaxFileSizeMB)
	}

	// Add walk context option if timeout is specified
	if a.cfg.Timeout > 0 {
		walkOptions = append(walkOptions, walker.WithContext(ctx))
	}

	// --- Define walk function ---
	printFunc := func(relativePath string, content []byte, err error) error {
		if err != nil {
			a.log.Warn("Skipping file '%s' due to error: %v", relativePath, err)
			return nil // Error handled by logging
		}

		// Debug: Log every file that reaches the printFunc
		a.log.Debug("Walk callback received file: %s (content nil? %v)",
			relativePath, content == nil)

		if content != nil { // Ensure content was actually read
			// Debug info before printing
			a.log.Debug("About to print file: %s (%d bytes)", relativePath, len(content))
			p.PrintFile(relativePath, content)
			// Debug info after printing
			a.log.Debug("After printing file: %s (printer count: %d)", relativePath, p.GetCount())
		} else {
			// This case shouldn't happen if err is nil, but good to log if it does
			a.log.Warn("printFunc called for '%s' with nil content and nil error.", relativePath)
		}
		return nil // Indicate success to walker
	}

	// --- Start the directory walk ---
	infoLog("Scanning directory: %s", absRootDir)
	if a.cfg.Concurrent {
		infoLog("Using concurrent processing with %d workers.", a.cfg.MaxWorkers)
	}

	skippedItems, err := walker.Walk(absRootDir, matcher, printFunc, walkOptions...)

	// --- Handle walk errors ---
	if err != nil {
		a.log.Error("Critical error during directory walk: %v", err)
		os.Exit(1)
	}

	// --- Show results summary ---
	infoLog("Found and processed %d files.", p.GetCount())

	// Finalize the printer (important for JSON output to close the array)
	p.Finalize()

	duration := time.Since(startTime)
	infoLog("Scan complete in %v.", duration.Round(time.Millisecond))

	// --- Show Skipped Items (if requested) ---
	if a.cfg.ShowSkipped {
		infoLog("--- Skipped Items (%d) ---", len(skippedItems))
		if len(skippedItems) > 0 {
			// Sort for consistent output
			sort.Slice(skippedItems, func(i, j int) bool {
				return skippedItems[i].Path < skippedItems[j].Path
			})
			for _, item := range skippedItems {
				typeStr := "FILE"
				if item.IsDir {
					typeStr = "DIR " // Add space for alignment
				}
				// Print to stderr
				fmt.Fprintf(os.Stderr, "Skipped %s: %-.*s [%s]\n",
					typeStr,
					50, // Max width for path column
					item.Path,
					item.Reason,
				)
			}
		} else {
			infoLog("No items were skipped.")
		}
		infoLog("--- End Skipped Items ---")
	}
}
