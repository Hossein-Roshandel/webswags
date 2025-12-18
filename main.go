package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/Hossein-Roshandel/webswags/discovery"
)

//go:embed templates/*
var templatesFS embed.FS

const (
	port             = "8085"
	swaggerUIVersion = "5.9.0"
	jsonFormat       = "json"
	yamlFormat       = "yaml"

	// Server timeouts in seconds.
	serverReadTimeout  = 15
	serverWriteTimeout = 15
	serverIdleTimeout  = 60

	// UI colors.
	colorJSON = "#f39c12" // Orange for JSON
	colorYAML = "#27ae60" // Green for YAML
)

var rootDir string //nolint:gochecknoglobals // Global variable to store root directory

// IndexData represents the data structure for the index page template.
type IndexData struct {
	TotalServices int
	Services      []discovery.SwaggerSpec
	Empty         bool
}

type ServiceData struct {
	ServiceTitle     string
	SwaggerUIVersion string
	Format           string
	FormatColor      string
	SpecURL          string
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		slog.Info("Request", "uri", r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Parse command line arguments
	flag.StringVar(&rootDir, "root", "..", "Root directory to search for swagger specifications")
	flag.Parse()

	slog.Info("Starting webswags server", "address", "http://localhost:"+port)
	slog.Info("Searching for specifications", "root_dir", rootDir)

	// Discover all swagger specs
	specs, err := discovery.DiscoverSwaggerSpecs(rootDir)
	if err != nil {
		slog.Error("Failed to discover swagger specs", "error", err)
		os.Exit(1)
	}

	slog.Info("Discovered swagger specifications", "count", len(specs))
	for _, spec := range specs {
		slog.Info("Service found", "name", spec.Name, "service", spec.Service)
	}

	// Setup routes.
	r := mux.NewRouter()

	// API routes.
	r.HandleFunc("/api/specs", handleSpecs(specs)).Methods("GET")
	r.HandleFunc("/api/specs/{service}/swagger.yaml", handleSwaggerFile(specs)).Methods("GET")
	r.HandleFunc("/api/specs/{service}/swagger.json", handleSwaggerFile(specs)).Methods("GET")

	// Main routes
	r.HandleFunc("/", handleIndex(specs)).Methods("GET")
	r.HandleFunc("/service/{service}", handleServiceSwagger(specs)).Methods("GET")

	r.Use(loggingMiddleware)

	slog.Info("Starting server", "address", "http://localhost:"+port)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  serverReadTimeout * time.Second,
		WriteTimeout: serverWriteTimeout * time.Second,
		IdleTimeout:  serverIdleTimeout * time.Second,
	}

	if serveErr := server.ListenAndServe(); serveErr != nil {
		slog.Error("Server failed to start", "error", serveErr)
		os.Exit(1)
	}
}

// handleIndex serves the main page listing all services.
func handleIndex(specs []discovery.SwaggerSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		tmpl, err := template.ParseFS(templatesFS, "templates/index.html")
		if err != nil {
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		data := IndexData{
			TotalServices: len(specs),
			Services:      specs,
			Empty:         len(specs) == 0,
		}

		if execErr := tmpl.Execute(w, data); execErr != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
	}
}

// handleServiceSwagger serves the Swagger UI for a specific service.
func handleServiceSwagger(specs []discovery.SwaggerSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		service := vars["service"]

		// Find the spec for this service to determine format
		specFormat := yamlFormat // default
		for _, spec := range specs {
			if spec.Service == service {
				specFormat = spec.Format
				break
			}
		}

		// Determine the correct URL based on format.
		specURL := fmt.Sprintf("/api/specs/%s/swagger.%s", service, specFormat)

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		tmpl, err := template.ParseFS(templatesFS, "templates/service.html")
		if err != nil {
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		data := ServiceData{
			ServiceTitle:     cases.Title(language.English, cases.Compact).String(service),
			SwaggerUIVersion: swaggerUIVersion,
			Format:           specFormat,
			FormatColor:      getFormatColor(specFormat),
			SpecURL:          specURL,
		}

		if execErr := tmpl.Execute(w, data); execErr != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
	}
}

// getFormatColor returns a color for the format badge.
func getFormatColor(format string) string {
	if format == jsonFormat {
		return colorJSON // Orange for JSON.
	}
	return colorYAML // Green for YAML.
}

// handleSpecs returns JSON list of all discovered specs.
func handleSpecs(specs []discovery.SwaggerSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if err := json.NewEncoder(w).Encode(specs); err != nil {
			http.Error(w, "Failed to encode specs to JSON", http.StatusInternalServerError)
			return
		}
	}
}

// handleSwaggerFile serves the raw YAML or JSON file for a specific service.
func handleSwaggerFile(specs []discovery.SwaggerSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		service := vars["service"]

		// Determine requested format from URL
		requestedFormat := yamlFormat
		if strings.HasSuffix(r.URL.Path, ".json") {
			requestedFormat = jsonFormat
		}

		// Find the spec for this service
		var specPath string
		var contentType string
		found := false

		for _, spec := range specs {
			if spec.Service == service && spec.Format == requestedFormat {
				specPath = spec.Path
				found = true
				break
			}
		}

		if !found {
			http.NotFound(w, r)
			return
		}

		if requestedFormat == jsonFormat {
			contentType = "application/json"
		} else {
			contentType = "text/yaml"
		}

		// Serve the file
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, specPath)
	}
}
