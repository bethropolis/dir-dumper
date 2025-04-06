// Package walker handles directory traversal and file processing
package walker

import (
	"context"
	"strings"

	"github.com/bethropolis/dir-dumper/internal/utils"
)

// WalkOptions configures the behavior of the Walk function
type WalkOptions struct {
	Logger       utils.Logger
	Concurrent   bool
	MaxWorkers   int
	MaxFileSize  int64
	ExtensionMap map[string]struct{}
	Context      context.Context
	ignoreHidden bool
	ProgressFn   ProgressCallback // Add progress callback function
}

// ProgressCallback is a function that receives progress updates
type ProgressCallback func(stats ProgressStats)

// ProgressStats holds statistics about the walk progress
type ProgressStats struct {
	TotalFiles      int64  // Total files seen
	ProcessedFiles  int64  // Files that passed all filters and were processed
	SkippedFiles    int64  // Files that were skipped for any reason
	TotalDirs       int64  // Total directories seen
	SkippedDirs     int64  // Directories that were skipped
	CurrentFilePath string // Path of the current file being processed (relative)
}

// defaultOptions returns the default walk options
func defaultOptions() WalkOptions {
	return WalkOptions{
		Logger:       &utils.NoopLogger{},
		Concurrent:   false,
		MaxWorkers:   10,
		MaxFileSize:  0,   // No limit
		ExtensionMap: nil, // No extension filtering by default
		Context:      context.Background(),
		ignoreHidden: false,
		ProgressFn:   nil,
	}
}

// Option is a functional option for configuring WalkOptions
type Option func(*WalkOptions)

// WithLogger sets a custom logger for the walker
func WithLogger(logger utils.Logger) Option {
	return func(opts *WalkOptions) {
		opts.Logger = logger
	}
}

// WithConcurrency enables or disables concurrent file processing
func WithConcurrency(enabled bool) Option {
	return func(opts *WalkOptions) {
		opts.Concurrent = enabled
	}
}

// WithMaxWorkers sets the maximum number of concurrent workers
func WithMaxWorkers(workers int) Option {
	return func(opts *WalkOptions) {
		if workers > 0 {
			opts.MaxWorkers = workers
		}
	}
}

// WithMaxFileSize sets the maximum file size to read in bytes
func WithMaxFileSize(maxBytes int64) Option {
	return func(opts *WalkOptions) {
		opts.MaxFileSize = maxBytes
	}
}

// WithExtensions sets the file extensions to include (without the dot)
func WithExtensions(extensions []string) Option {
	return func(opts *WalkOptions) {
		extMap := make(map[string]struct{}, len(extensions))
		for _, ext := range extensions {
			extMap[strings.TrimPrefix(ext, ".")] = struct{}{}
		}
		opts.ExtensionMap = extMap
	}
}

// WithExtensionMap sets the file extensions to include from a map (without the dot)
func WithExtensionMap(extMap map[string]struct{}) Option {
	return func(opts *WalkOptions) {
		if extMap != nil {
			newMap := make(map[string]struct{}, len(extMap))
			for ext := range extMap {
				newMap[strings.TrimPrefix(ext, ".")] = struct{}{}
			}
			opts.ExtensionMap = newMap
		} else {
			opts.ExtensionMap = nil
		}
	}
}

// WithContext sets the context for cancellation
func WithContext(ctx context.Context) Option {
	return func(opts *WalkOptions) {
		if ctx != nil {
			opts.Context = ctx
		}
	}
}

// WithIgnoreHidden enables or disables ignoring hidden files
func WithIgnoreHidden(enabled bool) Option {
	return func(opts *WalkOptions) {
		opts.ignoreHidden = enabled
	}
}

// WithProgress adds a progress callback function
func WithProgress(fn ProgressCallback) Option {
	return func(o *WalkOptions) {
		o.ProgressFn = fn
	}
}
