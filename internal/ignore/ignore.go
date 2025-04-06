// Package ignore provides file/directory pattern matching for exclusion
//
// This package handles advanced file/directory exclusion patterns based on
// multiple criteria including .gitignore rules, hidden files, and custom patterns.
// It uses the functional options pattern for configuration.
package ignore

// NewDefaultMatcher creates an IgnoreMatcher with default settings
func NewDefaultMatcher(rootDir string) (*IgnoreMatcher, error) {
	return New(rootDir)
}

// NewFromConfig creates an IgnoreMatcher from a Config struct
func NewFromConfig(cfg Config) (*IgnoreMatcher, error) {
	options := []Option{
		WithHiddenIgnore(cfg.IgnoreHidden),
		WithGitIgnore(cfg.IgnoreGit),
		WithRecursive(cfg.RecursiveMode),
		WithDisabled(cfg.Disabled),
	}

	if cfg.CustomRules != nil && len(cfg.CustomRules) > 0 {
		options = append(options, WithCustomRules(cfg.CustomRules))
	}

	if cfg.Logger != nil {
		options = append(options, WithLogger(cfg.Logger))
	}

	return New(cfg.RootDir, options...)
}

// CreateDisabledMatcher returns a matcher that ignores nothing
func CreateDisabledMatcher() *IgnoreMatcher {
	matcher, _ := New(".", WithDisabled(true))
	return matcher
}

// IsIgnored is a convenience function to check if a path should be ignored
func IsIgnored(matcher *IgnoreMatcher, path string, isDir bool) bool {
	if matcher == nil {
		return false
	}
	return matcher.ShouldIgnore(path, isDir)
}
