package ignore

import (
	"path/filepath"
	"strings"
)

// ShouldIgnore checks if a file or directory should be ignored
func (m *IgnoreMatcher) ShouldIgnore(relativePath string, isDir bool) bool {
	// Return early if matcher is nil or disabled
	if m == nil || m.disabled {
		return false
	}

	// Normalize empty paths
	if relativePath == "" || relativePath == "." {
		return false // Never ignore the root itself
	}

	m.logger.Debug("ignore.ShouldIgnore: Checking path: %q (isDir: %v)", relativePath, isDir)

	// Check for hidden files if ignoreHidden is enabled
	if m.ignoreHidden {
		// Check if the basename starts with a dot (more efficient than splitting)
		base := filepath.Base(relativePath)
		if strings.HasPrefix(base, ".") {
			m.logger.Debug("ignore.ShouldIgnore: Ignored %q (hidden file rule)", relativePath)
			return true
		}

		// Also check if any parent directory is hidden
		dir := filepath.Dir(relativePath)
		for dir != "." && dir != "/" && dir != "\\" {
			base = filepath.Base(dir)
			if strings.HasPrefix(base, ".") {
				m.logger.Debug("ignore.ShouldIgnore: Ignored %q (hidden dir rule)", relativePath)
				return true
			}
			dir = filepath.Dir(dir)
		}
	}

	// Special check for .git directory
	if m.ignoreGit && isPathInGitDir(relativePath, isDir) {
		m.logger.Debug("ignore.ShouldIgnore: Ignored %q (.git rule)", relativePath)
		return true
	}

	// Delegate to gitignore library
	if m.repoIgnore != nil {
		unixPath := filepath.ToSlash(relativePath) // Ensure forward slashes

		m.logger.Debug("ignore.ShouldIgnore: Checking library for path %q", unixPath) // Log before library call

		ignored := false
		included := false
		// Defensive wrapper for library calls
		func() {
			defer func() {
				if r := recover(); r != nil {
					m.logger.Error("PANIC recovered in gitignore library for path %q: %v", relativePath, r)
					// Treat panic as "cannot determine", maybe default to not ignoring? Or log and ignore?
					// Let's default to NOT ignoring if the library panics.
					ignored = false
					included = false
				}
			}()
			ignored = m.repoIgnore.Ignore(unixPath)
			if ignored {
				included = m.repoIgnore.Include(unixPath)
			}
		}() // End defensive func

		if ignored {
			m.logger.Debug("ignore.ShouldIgnore: Path %q ignored by library matcher", relativePath)
			if included {
				m.logger.Debug("ignore.ShouldIgnore: Path %q explicitly included by negation rule", relativePath)
				return false // Explicitly included, so NOT ignored
			}
			return true // Ignored and not explicitly included
		}
	} else {
		m.logger.Debug("ignore.ShouldIgnore: No repository ignore patterns loaded (m.repoIgnore is nil).", relativePath)
	}

	m.logger.Debug("ignore.ShouldIgnore: Path %q NOT ignored by any rule", relativePath)
	return false
}

// isPathInGitDir checks if a path is inside a .git directory
func isPathInGitDir(relativePath string, isDir bool) bool {
	parts := strings.Split(filepath.ToSlash(relativePath), "/")
	for i, part := range parts {
		if part == ".git" {
			// If .git is a directory component (not just a prefix of a filename)
			if isDir || i < len(parts)-1 {
				return true
			}
		}
	}
	return false
}
