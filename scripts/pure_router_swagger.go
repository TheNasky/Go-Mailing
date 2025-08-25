package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SwaggerSpec struct {
	Swagger string                 `json:"swagger"`
	Info    SwaggerInfo            `json:"info"`
	Host    string                 `json:"host"`
	Schemes []string               `json:"schemes"`
	Paths   map[string]interface{} `json:"paths"`
}

type SwaggerInfo struct {
	Version     string `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type RouteInfo struct {
	Module   string
	Prefix   string
	Path     string
	Method   string
	Handler  string
	FullPath string
}

func main() {
	fmt.Println("Generating swagger from router definitions only...")

	// Discover all routes from router files
	routes, err := discoverAllRoutes()
	if err != nil {
		log.Fatalf("Error discovering routes: %v", err)
	}

	fmt.Printf("Found %d routes\n", len(routes))

	// Generate swagger spec
	swagger := SwaggerSpec{
		Swagger: "2.0",
		Info: SwaggerInfo{
			Version:     "1.0",
			Title:       "Master Server API",
			Description: "API documentation generated from router definitions",
		},
		Host:    "localhost:8080",
		Schemes: []string{"http"},
		Paths:   make(map[string]interface{}),
	}

	// Add paths from routes
	for _, route := range routes {
		if swagger.Paths[route.FullPath] == nil {
			swagger.Paths[route.FullPath] = make(map[string]interface{})
		}

		pathMap := swagger.Paths[route.FullPath].(map[string]interface{})
		methodLower := strings.ToLower(route.Method)

		// Create method definition
		methodDef := map[string]interface{}{
			"summary":     fmt.Sprintf("%s %s", route.Method, route.FullPath),
			"description": fmt.Sprintf("Endpoint: %s", route.FullPath),
			"tags":        []string{route.Module},
			"produces":    []string{"application/json"},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success",
				},
			},
		}

		pathMap[methodLower] = methodDef
	}

	// Write swagger.json
	jsonBytes, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling swagger JSON: %v", err)
	}

	err = ioutil.WriteFile("docs/swagger.json", jsonBytes, 0644)
	if err != nil {
		log.Fatalf("Error writing swagger.json: %v", err)
	}

	fmt.Println("✓ Generated docs/swagger.json")
	fmt.Printf("✓ View at: http://localhost:8080/swagger/\n")
}

func discoverAllRoutes() ([]RouteInfo, error) {
	var allRoutes []RouteInfo

	// Walk through modules directory
	err := filepath.Walk("modules", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "router.go") {
			moduleName := filepath.Base(filepath.Dir(path))
			routes, err := parseRouterFile(path, moduleName)
			if err != nil {
				log.Printf("Warning: could not parse %s: %v", path, err)
				return nil
			}
			allRoutes = append(allRoutes, routes...)
		}
		return nil
	})

	return allRoutes, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseRouterFile(filename, moduleName string) ([]RouteInfo, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var routes []RouteInfo

	// Use a simpler approach: find all method calls with their prefixes
	// Look for patterns like: router.Router(r, "/prefix").Get("/path", handler)

	// For each method type, find the complete router.Router().Method() pattern
	methods := []string{"Get", "Post", "Put", "Delete", "Patch"}

	// Find all router.Router calls and their chained methods
	// Look for: router.Router(r, "/prefix").Method("/path", handler).Method("/path2", handler2)...

	// First, find all router.Router calls
	routerRe := regexp.MustCompile(`router\.Router\([^,]+,\s*"([^"]+)"\)`)
	routerMatches := routerRe.FindAllStringSubmatch(string(content), -1)

	fmt.Printf("  Found %d router.Router calls in %s\n", len(routerMatches), filename)

	for _, routerMatch := range routerMatches {
		if len(routerMatch) < 2 {
			continue
		}
		prefix := routerMatch[1]

		// Find the start position of this router.Router call
		routerStart := strings.Index(string(content), routerMatch[0])
		if routerStart == -1 {
			continue
		}

		// Look for method calls after this router.Router call
		// Find the next router.Router call or end of function to limit our search
		searchContent := string(content)[routerStart:]

		// Look for the end of the method chain - find the next semicolon or closing brace
		// that would indicate the end of the router.Router() chain
		nextRouterIndex := strings.Index(searchContent[1:], "router.Router(")
		semicolonIndex := strings.Index(searchContent, ";")
		closingBraceIndex := strings.Index(searchContent, "}")

		var searchEnd int
		if nextRouterIndex != -1 {
			searchEnd = nextRouterIndex + 1
		} else if semicolonIndex != -1 {
			searchEnd = semicolonIndex + 1
		} else if closingBraceIndex != -1 {
			searchEnd = closingBraceIndex
		} else {
			searchEnd = len(searchContent)
		}

		// Search within this scope for method calls
		scopeContent := searchContent[:searchEnd]

		// Look for chained method calls like Get("/path", handler).Post("/path2", handler2)
		for _, method := range methods {
			// Pattern: Method("/path", handler) - can be chained (no leading dot)
			// The methods are on separate lines, so we need to handle multiline content
			// Use (?s) flag to make . match newlines, and handle multiline content
			pattern := fmt.Sprintf(`(?s)%s\s*\(\s*"([^"]*)"\s*,\s*([^)]+)\s*\)`, method)
			re := regexp.MustCompile(pattern)
			matches := re.FindAllStringSubmatch(scopeContent, -1)

			for _, match := range matches {
				if len(match) > 2 {
					path := match[1]
					handler := strings.TrimSpace(match[2])

					// Build the full path
					fullPath := prefix
					if path != "" {
						if !strings.HasPrefix(path, "/") && fullPath != "/" {
							fullPath += "/"
						}
						fullPath += path
					}

					route := RouteInfo{
						Module:   moduleName,
						Prefix:   prefix,
						Path:     path,
						Method:   strings.ToUpper(method),
						Handler:  handler,
						FullPath: fullPath,
					}
					routes = append(routes, route)
				}
			}
		}
	}

	return routes, nil
}
