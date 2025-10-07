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

// DiscoverSwaggerSpecs scans within the given project root
// and returns a list of SwaggerSpec objects for all discovered Swagger files.
// It looks for files with .yaml or .json extensions inside any "spec" subdirectory.
//
// Example path structure:
//
//	connector/{service}/spec/{swagger_file}.yaml
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
			return nil //nolint:nilerr // Continue walking despite access errors
		}

		// Match files in a "spec" directory with .yaml or .json extensions
		if info.Mode().IsRegular() &&
			strings.Contains(path, string(filepath.Separator)+"spec"+string(filepath.Separator)) &&
			(strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".json")) {
			spec, err := parseSwaggerSpec(path)
			if err != nil {
				slog.Warn("Failed to parse swagger spec", "path", path, "error", err)
				return nil // Continue processing other files
			}
			specs = append(specs, spec)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking connector directory: %w", err)
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
		Service:  deriveName("", path),
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
		spec.Name = deriveName(spec.Title, spec.FileName)

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
		spec.Name = deriveName(spec.Title, spec.FileName)

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

// deriveName returns a title-cased service name.
// Priority:
//  1. If title (after trimming) is non-empty, it is used.
//  2. Otherwise, the name is derived from path by taking the path segment
//     immediately preceding the first "spec" directory (using the OS path separator).
//  3. If no such "spec" segment exists (or it has no preceding segment),
//     the filename (without extension) is used.
//
// The final value is title-cased using English rules.
//
// Examples:
//
//	deriveName("  loyalty  ", "connector/loyalty/spec/loyalty.yaml") => "Loyalty"  // title wins
//	deriveName("",            "connector/loyalty/spec/loyalty.yaml") => "Loyalty"  // segment before "spec"
//	deriveName("",            "foo/bar/spec/x.yaml")                  => "Bar"     // segment before "spec"
//	deriveName("",            "foo/bar/baz.yaml")                     => "Baz"     // fallback to filename w/o ext
//
// Notes:
//   - Only the segment before "spec" is used when present; the actual filename is ignored in that case.
//   - If "spec" is the first path segment, the filename (without extension) is used instead of returning empty.
//   - Title-casing uses cases.Title(language.English, cases.Compact), which may alter
//     acronyms/branding (e.g., "HEllo" -> "Hello").
//   - Path splitting respects the OS-specific separator (filepath.Separator).
func deriveName(title, path string) string {
	var serviceName = strings.TrimSpace(title)
	if serviceName != "" {
		return cases.Title(language.English, cases.Compact).String(serviceName)
	}

	// Normalize and prepare filename fallback (without extension).
	cleanPath := filepath.Clean(path)
	base := filepath.Base(cleanPath)
	if base == "." || base == string(filepath.Separator) {
		base = ""
	}
	filenameNoExt := strings.TrimSuffix(base, filepath.Ext(base))

	// Try to get the segment immediately preceding "spec".
	pathParts := strings.Split(cleanPath, string(filepath.Separator))
	serviceName = filenameNoExt // default fallback

	for i := range len(pathParts) - 1 {
		if pathParts[i] == "spec" && pathParts[i-1] != "" {
			serviceName = pathParts[i-1]
			break
		}
	}
	return cases.Title(language.English, cases.Compact).String(serviceName)
}
