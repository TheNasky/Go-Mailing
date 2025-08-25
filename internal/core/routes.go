package core

import (
	"fmt"
	"net/http"

	"github.com/thenasky/go-framework/internal/logger"

	"github.com/gorilla/mux"
)

// ModuleRegistrar interface that modules must implement for auto-registration
type ModuleRegistrar interface {
	RegisterRoutes(r *mux.Router)
}

// ModuleInfo holds information about a discovered module
type ModuleInfo struct {
	Name   string
	Module ModuleRegistrar
}

// discoveredModules holds all automatically discovered modules
var discoveredModules []ModuleInfo

func NewRouter() http.Handler {
	router := mux.NewRouter()

	// Automatically discover and register all modules
	discoverModules()

	// Register all discovered modules
	for _, moduleInfo := range discoveredModules {
		moduleInfo.Module.RegisterRoutes(router)
	}

	// Swagger documentation - serve our custom swagger.json
	router.HandleFunc("/swagger", swaggerUIHandler).Methods("GET")
	router.HandleFunc("/swagger/", swaggerUIHandler).Methods("GET")
	router.HandleFunc("/swagger/swagger.json", swaggerJSONHandler).Methods("GET")

	// Custom 404 handler
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	// Apply middleware
	return logger.RequestLogger(router)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	// Log the 404 error with the custom tag
	logger.LogNotFound(fmt.Sprintf("Route not found: %s %s", r.Method, r.URL.Path))
}

// swaggerUIHandler serves a simple Swagger UI HTML page
func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/swagger/swagger.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// swaggerJSONHandler serves the swagger.json file
func swaggerJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	http.ServeFile(w, r, "docs/swagger.json")
}

// discoverModules automatically finds and loads all modules in the modules/ directory
func discoverModules() {
	if len(discoveredModules) > 0 {
		return // Already discovered
	}

	// Load all registered modules from the registry
	for moduleName, module := range moduleRegistry {
		discoveredModules = append(discoveredModules, ModuleInfo{
			Name:   moduleName,
			Module: module,
		})
	}

}

// moduleRegistry holds all available modules
var moduleRegistry = make(map[string]ModuleRegistrar)

// RegisterModule allows modules to register themselves
func RegisterModule(name string, module ModuleRegistrar) {
	moduleRegistry[name] = module
}
