package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
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

	// Proxy configuration.
	proxyTimeout    = 30        // seconds
	proxyBufferSize = 32 * 1024 // 32KB buffer for streaming

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

	// CORS proxy route - allows Swagger UI to make requests through our server
	r.HandleFunc("/proxy", handleProxy()).Methods(
		"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "CONNECT", "TRACE",
	)

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

		tmpl, err := template.ParseFS(
			templatesFS,
			"templates/index.html",
			"templates/index-styles.css",
			"templates/theme.css",
			"templates/theme.js",
			"templates/SwaggerDark.css",
		)
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
			slog.Error("Failed to render index template", "error", execErr)
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

		tmpl, err := template.ParseFS(
			templatesFS,
			"templates/service.html",
			"templates/service-styles.css",
			"templates/service-script.js",
			"templates/theme.css",
			"templates/theme.js",
			"templates/SwaggerDark.css",
		)
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
			slog.Error("Failed to render service template", "service", service, "error", execErr)
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

// setCORSHeaders sets CORS headers on the response writer.
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set(
		"Access-Control-Allow-Methods",
		"GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE",
	)
	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Content-Type, Authorization, X-Requested-With, Accept, X-API-Key, X-Custom-Header",
	)
}

// handlePreflightRequest handles CORS preflight OPTIONS requests.
func handlePreflightRequest(w http.ResponseWriter) {
	setCORSHeaders(w)
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.WriteHeader(http.StatusOK)
}

// copyRequestHeaders copies headers from the original request to the proxy request, excluding Host.
func copyRequestHeaders(proxyReq *http.Request, originalReq *http.Request) {
	for key, values := range originalReq.Header {
		if key != "Host" {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}
	}
}

// copyQueryParameters copies query parameters from the original request to the proxy request,
// excluding the 'url' parameter.
func copyQueryParameters(proxyReq *http.Request, originalReq *http.Request) {
	query := originalReq.URL.Query()
	query.Del("url") // Remove the proxy URL parameter
	proxyReq.URL.RawQuery = query.Encode()
}

// copyResponseHeaders copies headers from the proxy response to the response writer.
func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

// streamResponseBody streams the response body from the proxy response to the response writer.
func streamResponseBody(w http.ResponseWriter, resp *http.Response) error {
	buf := make([]byte, proxyBufferSize)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write response: %w", writeErr)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read response: %w", readErr)
		}
	}
	return nil
}

// handleProxy acts as a CORS proxy for API requests made from Swagger UI.
// It forwards requests to the actual API servers, bypassing CORS restrictions.
// Usage: /proxy?url={target-url}
// Example: /proxy?url=https://testcertsapi.bpglobal.com/VEDAUTH/Authorize/OAuth
func handleProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the target URL from query parameter
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Error(w, "Target URL is required. Usage: /proxy?url={target-url}", http.StatusBadRequest)
			return
		}

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			handlePreflightRequest(w)
			return
		}

		// Create the proxied request with context
		proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
		if err != nil {
			slog.Error("Failed to create proxy request", "error", err, "target_url", targetURL)
			http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
			return
		}

		// Copy headers and query parameters
		copyRequestHeaders(proxyReq, r)
		copyQueryParameters(proxyReq, r)

		// Make the request
		client := &http.Client{
			Timeout: proxyTimeout * time.Second,
		}
		resp, err := client.Do(proxyReq)
		if err != nil {
			slog.Error("Proxy request failed", "error", err, "target_url", targetURL)
			http.Error(w, fmt.Sprintf("Proxy request failed: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Set CORS headers and copy response headers
		setCORSHeaders(w)
		copyResponseHeaders(w, resp)

		// Set status code
		w.WriteHeader(resp.StatusCode)

		// Stream the response body
		if streamErr := streamResponseBody(w, resp); streamErr != nil {
			slog.Error("Failed to write proxy response", "error", streamErr)
			return
		}

		slog.Info("Proxy request completed", "method", r.Method, "target_url", targetURL, "status", resp.StatusCode)
	}
}
