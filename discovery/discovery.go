package discovery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	oas3 "github.com/getkin/kin-openapi/openapi3"
	oas2 "github.com/go-openapi/spec"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"sigs.k8s.io/yaml"
)

// ---------- Shared leaf types (expanded Info is recommended) ----------

type Info struct {
	Title          string   `json:"title"                    yaml:"title"`
	Version        string   `json:"version"                  yaml:"version"`
	Description    string   `json:"description,omitempty"    yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"        yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty"        yaml:"license,omitempty"`
}

type Contact struct {
	Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
	URL   string `json:"url,omitempty"   yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type License struct {
	Name string `json:"name"          yaml:"name"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// ---------- Version-specific "views" (lightweight, generic) ----------

// Swagger2Doc captures top-level Swagger 2.0 fields.
// Large/variable subtrees stay as json.RawMessage to remain schema-agnostic here.
type Swagger2Doc struct {
	Swagger             string          `json:"swagger"                       yaml:"swagger"` // e.g., "2.0"
	Info                Info            `json:"info"                          yaml:"info"`
	Host                string          `json:"host,omitempty"                yaml:"host,omitempty"`
	BasePath            string          `json:"basePath,omitempty"            yaml:"basePath,omitempty"`
	Schemes             []string        `json:"schemes,omitempty"             yaml:"schemes,omitempty"`
	Consumes            []string        `json:"consumes,omitempty"            yaml:"consumes,omitempty"`
	Produces            []string        `json:"produces,omitempty"            yaml:"produces,omitempty"`
	Paths               json.RawMessage `json:"paths,omitempty"               yaml:"paths,omitempty"`
	Definitions         json.RawMessage `json:"definitions,omitempty"         yaml:"definitions,omitempty"`
	Parameters          json.RawMessage `json:"parameters,omitempty"          yaml:"parameters,omitempty"`
	Responses           json.RawMessage `json:"responses,omitempty"           yaml:"responses,omitempty"`
	SecurityDefinitions json.RawMessage `json:"securityDefinitions,omitempty" yaml:"securityDefinitions,omitempty"`
	Security            json.RawMessage `json:"security,omitempty"            yaml:"security,omitempty"`
	Tags                json.RawMessage `json:"tags,omitempty"                yaml:"tags,omitempty"`
	ExternalDocs        json.RawMessage `json:"externalDocs,omitempty"        yaml:"externalDocs,omitempty"`
}

// OpenAPI3Doc captures top-level OpenAPI 3.x/3.1 fields.
type OpenAPI3Doc struct {
	OpenAPI      string          `json:"openapi"                     yaml:"openapi"` // e.g., "3.0.3" / "3.1.0"
	Info         Info            `json:"info"                        yaml:"info"`
	Servers      json.RawMessage `json:"servers,omitempty"           yaml:"servers,omitempty"`
	Paths        json.RawMessage `json:"paths,omitempty"             yaml:"paths,omitempty"`
	Components   json.RawMessage `json:"components,omitempty"        yaml:"components,omitempty"`
	Security     json.RawMessage `json:"security,omitempty"          yaml:"security,omitempty"`
	Tags         json.RawMessage `json:"tags,omitempty"              yaml:"tags,omitempty"`
	ExternalDocs json.RawMessage `json:"externalDocs,omitempty"      yaml:"externalDocs,omitempty"`
	// OAS 3.1 additions:
	JSONSchemaDialect string          `json:"jsonSchemaDialect,omitempty" yaml:"jsonSchemaDialect,omitempty"`
	Webhooks          json.RawMessage `json:"webhooks,omitempty"          yaml:"webhooks,omitempty"`
}

// ---------- Unified container ----------

// SwaggerSpec is a complete, version-agnostic container for API specs.
// Exactly one of V2 or V3 should be non-nil. Both are embedded so the present one inlines on marshal.
// The canonical documents (DocV2/DocV3) preserve the full, typed content.
type SwaggerSpec struct {
	// --- Version-specific top-level fields (one of these will be set) ---
	*Swagger2Doc `json:",omitempty" yaml:",omitempty,inline"`
	*OpenAPI3Doc `json:",omitempty" yaml:",omitempty,inline"`

	// --- Metadata (yours; keep/extend as needed) ---
	Name        string `json:"name"        yaml:"name"`
	Title       string `json:"title"       yaml:"title"`
	Version     string `json:"version"     yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Path        string `json:"path"        yaml:"path"`
	Service     string `json:"service"     yaml:"service"`
	Format      string `json:"format"      yaml:"format"` // "yaml" or "json"
	FileName    string `json:"fileName"    yaml:"fileName"`

	// --- Version markers (redundant but handy for quick checks) ---
	OpenAPIVersion string `json:"openapiVersion,omitempty" yaml:"openapiVersion,omitempty"` // e.g., "3.1.0"
	SwaggerVersion string `json:"swaggerVersion,omitempty" yaml:"swaggerVersion,omitempty"` // e.g., "2.0"

	// --- Canonical, full documents (for complete fidelity & downstream logic) ---
	DocV3 *oas3.T       `json:"-" yaml:"-"` // full OpenAPI 3.x/3.1 document
	DocV2 *oas2.Swagger `json:"-" yaml:"-"` // full Swagger 2.0 document

	// (Optional) Keep raw bytes if you need to re-serve the original file as-is.
	Raw []byte `json:"-" yaml:"-"`
}

// DiscoverSwaggerSpecs scans within the given project root recursively
// and returns a list of SwaggerSpec objects for all discovered OpenAPI/Swagger files.
// It looks for files with .yaml, .yml, or .json extensions anywhere under the root path.
//
// The function attempts to parse each YAML/JSON file as an OpenAPI/Swagger spec.
// Files that cannot be parsed or are not valid OpenAPI specs are skipped with a warning.
//
// Example path structures supported:
//   - connector/{service}/spec/{swagger_file}.yaml
//   - apis/{service}/{swagger_file}.json
//   - docs/openapi.yaml
//   - any/nested/path/to/spec.yml
//
// Parameters:
//   - projectRoot: the root directory of the project.
//
// Returns:
//   - A sorted slice of SwaggerSpec objects by service name.
//   - An error if the directory walk fails.
func DiscoverSwaggerSpecs(projectRoot string) ([]SwaggerSpec, error) {
	var specs []SwaggerSpec

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Skip this file or directory if there's an error accessing it
			slog.Debug("Skipping path due to access error", "path", path, "error", walkErr)
			return nil // Continue walking despite access errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file has a valid OpenAPI/Swagger extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}

		// Try to parse as an OpenAPI/Swagger spec
		spec, err := parseSwaggerSpec(path)
		if err != nil {
			// Only log at debug level since many YAML/JSON files won't be OpenAPI specs
			slog.Debug("Skipping file (not a valid OpenAPI spec)", "path", path, "error", err)
			return nil // Continue processing other files
		}

		specs = append(specs, spec)
		slog.Info("Discovered OpenAPI spec", "service", spec.Service, "version", spec.Version, "path", path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	// Sort specs alphabetically by service name
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Service < specs[j].Service
	})

	return specs, nil
}

// parseSwaggerSpec parses a YAML or JSON file and extracts OpenAPI/Swagger information.
// It tries OpenAPI 3.x/3.1 first (kin-openapi), then falls back to Swagger 2.0 (go-openapi/spec).
func parseSwaggerSpec(path string) (SwaggerSpec, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return SwaggerSpec{}, fmt.Errorf("failed to read file: %w", err)
	}

	spec := SwaggerSpec{
		Path:     path,
		FileName: filepath.Base(path),
		Format:   detectFormatFromExtOrContent(path, data),
		Raw:      data,
	}

	// --- Try OpenAPI 3.x/3.1 first using kin-openapi ---
	loader := &oas3.Loader{IsExternalRefsAllowed: true}
	if doc3, err3 := loader.LoadFromData(data); err3 == nil && doc3 != nil && strings.TrimSpace(doc3.OpenAPI) != "" {
		spec.DocV3 = doc3
		spec.OpenAPIVersion = strings.TrimSpace(doc3.OpenAPI)

		// Fill version-specific view (inlined when marshalled)
		spec.OpenAPI3Doc = &OpenAPI3Doc{
			OpenAPI:      spec.OpenAPIVersion,
			Info:         toInfoFromOAS3(doc3.Info),
			Servers:      toRaw(doc3.Servers),
			Paths:        toRaw(doc3.Paths),
			Components:   toRaw(doc3.Components),
			Security:     toRaw(doc3.Security),
			Tags:         toRaw(doc3.Tags),
			ExternalDocs: toRaw(doc3.ExternalDocs),
		}

		// Fill metadata summary (version-agnostic)
		spec.Title = spec.OpenAPI3Doc.Info.Title
		spec.Version = spec.OpenAPI3Doc.Info.Version
		spec.Description = spec.OpenAPI3Doc.Info.Description
		spec.Name = deriveName(spec.Title, path)
		spec.Service = deriveName(spec.Title, path)

		return spec, nil
	}

	// --- Fall back to Swagger 2.0 using go-openapi/spec ---
	var doc2 oas2.Swagger
	if err2 := unmarshalYAMLOrJSON(data, &doc2); err2 == nil && strings.TrimSpace(doc2.Swagger) != "" {
		spec.DocV2 = &doc2
		spec.SwaggerVersion = strings.TrimSpace(doc2.Swagger)

		spec.Swagger2Doc = &Swagger2Doc{
			Swagger:             spec.SwaggerVersion,
			Info:                toInfoFromOAS2(doc2.Info),
			Host:                doc2.Host,
			BasePath:            doc2.BasePath,
			Schemes:             doc2.Schemes,
			Consumes:            doc2.Consumes,
			Produces:            doc2.Produces,
			Paths:               toRaw(doc2.Paths),
			Definitions:         toRaw(doc2.Definitions),
			Parameters:          toRaw(doc2.Parameters),
			Responses:           toRaw(doc2.Responses),
			SecurityDefinitions: toRaw(doc2.SecurityDefinitions),
			Security:            toRaw(doc2.Security),
			Tags:                toRaw(doc2.Tags),
			ExternalDocs:        toRaw(doc2.ExternalDocs),
		}

		spec.Title = spec.Swagger2Doc.Info.Title
		spec.Version = spec.Swagger2Doc.Info.Version
		spec.Description = spec.Swagger2Doc.Info.Description
		spec.Name = deriveName(spec.Title, path)
		spec.Service = deriveName(spec.Title, path)

		return spec, nil
	}

	return SwaggerSpec{}, fmt.Errorf("file %q is not a valid OpenAPI 3.x/3.1 or Swagger 2.0 document", path)
}

// --- Helpers ---

// detectFormatFromExtOrContent determines the file format (yaml or json) based on file extension
// or by examining the content if the extension is ambiguous.
func detectFormatFromExtOrContent(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		if looksLikeJSON(data) {
			return "json"
		}
		return "yaml"
	}
}

// by examining if it starts with '{' or '[' after trimming whitespace.
func looksLikeJSON(data []byte) bool {
	trim := strings.TrimLeftFunc(string(data), func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
	return strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "[")
}

// unmarshalYAMLOrJSON unmarshals YAML or JSON data into the provided interface.
// It first checks if the data looks like JSON, and if so, unmarshals directly.
// Otherwise, it converts YAML to JSON first, then unmarshals.
func unmarshalYAMLOrJSON(data []byte, v any) error {
	if looksLikeJSON(data) {
		return json.Unmarshal(data, v)
	}
	j, err := yaml.YAMLToJSON(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, v)
}

// toRaw safely marshals any value into a json.RawMessage.
// Returns nil if the value is nil, empty, or marshals to "null".
func toRaw(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return nil
	}
	return json.RawMessage(b)
}

// toInfoFromOAS3 converts a kin-openapi Info struct to our internal Info struct.
// It extracts and trims the title, version, description, terms of service,
// and contact/license information from the OpenAPI 3.x specification.
func toInfoFromOAS3(in *oas3.Info) Info {
	if in == nil {
		return Info{}
	}
	out := Info{
		Title:          strings.TrimSpace(in.Title),
		Version:        strings.TrimSpace(in.Version),
		Description:    strings.TrimSpace(in.Description),
		TermsOfService: strings.TrimSpace(in.TermsOfService),
	}
	if in.Contact != nil {
		out.Contact = &Contact{
			Name:  strings.TrimSpace(in.Contact.Name),
			Email: strings.TrimSpace(in.Contact.Email),
			URL:   strings.TrimSpace(in.Contact.URL),
		}
	}
	if in.License != nil {
		out.License = &License{
			Name: strings.TrimSpace(in.License.Name),
			URL:  strings.TrimSpace(in.License.URL),
		}
	}
	return out
}

// toInfoFromOAS2 converts a go-openapi/spec Info struct to our internal Info struct.
// It extracts and trims the title, version, description, terms of service,
// and contact/license information from the Swagger 2.0 specification.
func toInfoFromOAS2(in *oas2.Info) Info {
	if in == nil {
		return Info{}
	}
	out := Info{
		Title:          strings.TrimSpace(in.Title),
		Version:        strings.TrimSpace(in.Version),
		Description:    strings.TrimSpace(in.Description),
		TermsOfService: strings.TrimSpace(in.TermsOfService),
	}
	if in.Contact != nil {
		out.Contact = &Contact{
			Name:  strings.TrimSpace(in.Contact.Name),
			URL:   strings.TrimSpace(in.Contact.URL),
			Email: strings.TrimSpace(in.Contact.Email),
		}
	}
	if in.License != nil {
		out.License = &License{
			Name: strings.TrimSpace(in.License.Name),
			URL:  strings.TrimSpace(in.License.URL),
		}
	}
	return out
}

// deriveName returns a title-cased service name with intelligent fallback logic.
//
// Priority (first non-empty value wins):
//  1. If title (after trimming) is non-empty, it is used.
//  2. Extract from path using one of these strategies (in order):
//     a. Segment immediately before "spec" directory
//     b. Segment immediately before "api", "apis", "swagger", or "openapi" directory
//     c. Parent directory name (if not ".", root, or generic names)
//     d. Filename without extension
//
// The final value is title-cased using English rules.
//
// Examples:
//
//	deriveName("My API", "connector/loyalty/spec/loyalty.yaml")        => "My API"         // title wins
//	deriveName("",        "connector/loyalty/spec/loyalty.yaml")       => "Loyalty"        // segment before "spec"
//	deriveName("",        "apis/user-service/openapi.yaml")            => "User Service"   // segment before "apis"
//	deriveName("",        "docs/petstore/swagger.json")                => "Petstore"       // parent dir
//	deriveName("",        "/tmp/my-api.yaml")                          => "My Api"         // filename fallback
//	deriveName("",        "openapi.yaml")                              => "Openapi"        // filename only
//
// Notes:
//   - Hyphens and underscores in names are replaced with spaces before title-casing.
//   - Generic directory names like "docs", "specifications", etc. are skipped.
//   - Title-casing uses cases.Title(language.English, cases.Compact).
func deriveName(title, path string) string {
	serviceName := strings.TrimSpace(title)
	if serviceName != "" {
		return formatServiceName(serviceName)
	}

	// Normalize path and prepare fallbacks
	cleanPath := filepath.Clean(path)
	pathParts := strings.Split(cleanPath, string(filepath.Separator))

	// Filename without extension (last resort fallback)
	base := filepath.Base(cleanPath)
	filenameNoExt := strings.TrimSuffix(base, filepath.Ext(base))

	// Generic directory names to skip when looking for service name
	genericDirs := map[string]bool{
		"docs":           true,
		"doc":            true,
		"documentation":  true,
		"specifications": true,
		"specs":          true,
		".":              true,
		"/":              true,
		"":               true,
	}

	// Strategy 1: Look for segment before common API directory names
	apiDirNames := []string{"spec", "specs", "api", "apis", "swagger", "openapi", "oas"}
	for i := len(pathParts) - 1; i > 0; i-- {
		currentPart := strings.ToLower(pathParts[i])
		for _, apiDir := range apiDirNames {
			if currentPart == apiDir && i > 0 {
				candidate := pathParts[i-1]
				if !genericDirs[strings.ToLower(candidate)] {
					return formatServiceName(candidate)
				}
			}
		}
	}

	// Strategy 2: Use parent directory if it's not generic
	const minPathDepth = 2
	if len(pathParts) >= minPathDepth {
		parentDir := pathParts[len(pathParts)-2]
		if !genericDirs[strings.ToLower(parentDir)] {
			return formatServiceName(parentDir)
		}
	}

	// Strategy 3: Fallback to filename without extension
	return formatServiceName(filenameNoExt)
}

// formatServiceName formats a raw service name for display.
// It replaces hyphens and underscores with spaces, then applies title casing.
//
// Examples:
//   - "user-service" => "User Service"
//   - "my_api_v2" => "My Api V2"
//   - "UserAPI" => "Userapi"  (limitation of title casing)
func formatServiceName(name string) string {
	// Replace common separators with spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Clean up multiple spaces
	name = strings.Join(strings.Fields(name), " ")

	// Apply title casing
	return cases.Title(language.English, cases.Compact).String(name)
}
