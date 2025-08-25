package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/thenasky/go-framework/internal/core"
	"github.com/thenasky/go-framework/internal/database"
	"github.com/thenasky/go-framework/internal/logger"

	// Import modules for auto-registration (init functions)
	_ "github.com/thenasky/go-framework/modules/demo"
	_ "github.com/thenasky/go-framework/modules/email"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using default settings")
	}

	// Auto-generate swagger documentation
	generateSwaggerDocs()

	// Connect to MongoDB first
	logger.LogInfo("Connecting to MongoDB...")
	database.ConnectMongoDB()

	// Wait a moment for MongoDB connection to establish
	time.Sleep(2 * time.Second)

	// Now create router (this will initialize email module)
	router := core.NewRouter()

	logger.LogInfo("Server running at http://localhost:8080...")
	if err := http.ListenAndServe("localhost:8081", router); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

// generateSwaggerDocs generates swagger purely from router definitions
func generateSwaggerDocs() {
	// Check if swagger docs need regeneration
	if !shouldRegenerateSwagger() {
		return
	}

	// Run pure router swagger generator (silently)
	cmd := exec.Command("go", "run", "../scripts/pure_router_swagger.go")
	cmd.Dir = "."

	// Generate silently, only log errors
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogError("Failed to generate swagger: " + string(output))
		return
	}
}

// shouldRegenerateSwagger checks if router files are newer than generated docs
func shouldRegenerateSwagger() bool {
	docsFile := "docs/swagger.json"

	// If docs don't exist, generate them
	docsInfo, err := os.Stat(docsFile)
	if err != nil {
		return true
	}

	docsModTime := docsInfo.ModTime()

	// Only check router files (not controllers - they don't matter anymore)
	var needsRegeneration bool
	filepath.Walk("modules", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors, continue walking
		}

		// Only check router.go files
		if strings.HasSuffix(path, "router.go") {
			if info.ModTime().After(docsModTime) {
				needsRegeneration = true
				return filepath.SkipAll // we can stop walking once we find one
			}
		}
		return nil
	})

	return needsRegeneration
}
