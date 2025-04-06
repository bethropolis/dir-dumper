package ignore

import (
	"fmt"
	"path/filepath"

	"github.com/bethropolis/dir-dumper/internal/utils"
	gitignore "github.com/denormal/go-gitignore"
)

// New creates and initializes an IgnoreMatcher
func New(rootDir string, opts ...Option) (*IgnoreMatcher, error) {
	// First, get the absolute path for the root directory
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("ignore: failed to get absolute path for rootDir '%s': %w", rootDir, err)
	}

	// Initialize with default configuration
	matcher := &IgnoreMatcher{
		rootDir:       absRootDir,
		ignoreHidden:  true, // Default
		ignoreGit:     true, // Default
		recursiveMode: true, // Default
		logger:        &utils.NoopLogger{},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(matcher)
	}

	// Initialize the gitignore engine
	if err := matcher.init(); err != nil {
		return nil, err
	}

	return matcher, nil
}

// init initializes the gitignore engine
func (m *IgnoreMatcher) init() error {
	m.logger.Debug("ignore.New: Initializing for root: %s", m.rootDir)
	m.logger.Debug("ignore.New: ignoreHidden flag set to: %v", m.ignoreHidden)
	m.logger.Debug("ignore.New: ignoreGit flag set to: %v", m.ignoreGit)

	// Skip gitignore initialization if the matcher is disabled
	if m.disabled {
		m.logger.Debug("ignore.New: Matcher is disabled, skipping gitignore initialization")
		return nil
	}

	// Always use the repository approach to load gitignore files recursively
	// This better matches git's actual behavior
	repoMatcher, repoErr := gitignore.NewRepository(m.rootDir)

	if repoErr != nil {
		m.logger.Warn("ignore.New: Error loading repository ignores from '%s': %v", m.rootDir, repoErr)
		// Check if it's just that no ignore files were found
		if repoMatcher == nil {
			m.logger.Warn("ignore.New: No .gitignore files found or loaded by library for '%s'. Continuing without repo rules.", m.rootDir)
			// Create an empty matcher so methods don't panic
			repoMatcher = gitignore.New(nil, "", nil)
		} else {
			// If there was a more serious error during loading
			return fmt.Errorf("ignore: failed to load repository ignores: %w", repoErr)
		}
	}
	m.repoIgnore = repoMatcher
	m.logger.Debug("ignore.New: Successfully loaded repository ignores.")

	// Explicitly ignore .git directory if the flag requires it
	if m.ignoreGit {
		m.logger.Debug("ignore.New: Explicitly adding /.git/ pattern.")
		// Add to custom patterns
		m.customPatterns = append(m.customPatterns, "/.git/")
	}

	return nil
}
