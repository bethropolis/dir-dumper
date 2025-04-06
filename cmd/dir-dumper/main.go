package main

import (
	"os"

	"github.com/bethropolis/dir-dumper/internal/app"
	"github.com/bethropolis/dir-dumper/internal/config"
)

func main() {
	// Load configuration from command-line flags
	cfg := config.New()

	// Create and run the application
	application := app.New(cfg)

	// Run the application
	application.Run()

	// Close output file if one was opened
	if cfg.OutputFile != "" {
		if f, ok := application.Output.(*os.File); ok {
			f.Close()
		}
	}
}
