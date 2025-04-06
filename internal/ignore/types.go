// Package ignore provides file/directory pattern matching for exclusion
package ignore

import (
	"github.com/bethropolis/dir-dumper/internal/utils"
	gitignore "github.com/denormal/go-gitignore"
)

// IgnoreMatcher determines whether a file or directory should be ignored
type IgnoreMatcher struct {
	// The core gitignore object handling repository rules
	repoIgnore gitignore.GitIgnore

	// Configuration flags
	rootDir        string
	ignoreHidden   bool
	ignoreGit      bool
	recursiveMode  bool
	customPatterns []string
	logger         utils.Logger
	disabled       bool
}

// Config holds configuration options for the ignore matcher
type Config struct {
	RootDir       string
	IgnoreHidden  bool
	IgnoreGit     bool
	RecursiveMode bool
	CustomRules   []string
	Logger        utils.Logger
	Disabled      bool
}
