# WebSwags - API Documentation Server

WebSwags is a lightweight Go web server that provides a modern web interface for viewing OpenAPI/Swagger specifications in your projects.

## Features

- 🔍 **Recursive Auto-Discovery**: Walks the entire `-root` directory tree to find OpenAPI/Swagger specs (YAML/YML/JSON) in any folder structure.
- 📁 **Dual Format Support**: Parses both OpenAPI 3.x and Swagger 2.0 definitions regardless of YAML or JSON format.
- 🌐 **Modern UI**: Clean, responsive interface powered by Swagger UI 5.x with live theme toggling (light/dark/system).
- 🏷️ **Format Indicators**: Visual badges and version tags so you can spot YAML vs JSON (and their versions) instantly.
- 📋 **Rich Service Listing**: Hero stats, two-line descriptions, and quick-launch links for every discovered spec.
- � **Viewer Toggle**: Switch between Swagger UI and Redoc with a single click per service page.
- � **Proxy Controls**: Built-in, user-toggleable CORS proxy with clear ON/OFF state and warnings when running direct.
- 🎯 **Development Focus**: Designed specifically for local workflows—no external services required.

## Installation

### Prerequisites

- Go 1.25 or later

### Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/Hossein-Roshandel/webswags.git
   cd webswags
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. (Optional) Set up development environment:

   ```bash
   ./setup-dev.sh
   ```

## Usage

### Start the Server

```bash
# Build and run
go run main.go -root /path/to/project/root
```

### Command Line Options

- `-root <directory>`: Root directory to search for swagger specifications (default: "..")

Example:

```bash
# Use current directory as root
go run main.go -root $(pwd)

# Use specific directory
go run main.go -root /home/user/my-project
```

### Access the Documentation

1. Open [http://localhost:8085](http://localhost:8085) in your browser
2. Browse the list of available API services
3. Click on any service to view its Swagger documentation

## Architecture

### File Structure

```bash
webswags/
├── main.go              # Main server application
├── go.mod              # Go module definition with dependencies
├── go.sum              # Dependency checksums
├── discovery/
│   └── discovery.go    # Spec discovery and parsing logic
├── templates/
│   ├── index.html            # Service listing page template
│   ├── index-styles.css      # Landing page styles
│   ├── service.html          # Individual service page template
│   ├── service-styles.css    # Swagger/Redoc specific styles
│   ├── service-script.js     # Proxy + viewer toggle logic
│   ├── theme.css             # Shared light/dark theme tokens
│   ├── theme.js              # Theme toggle + dark-mode wiring
│   └── SwaggerDark.css       # Dark-theme overrides for Swagger UI
└── README.md           # This documentation
```

### Discovery Logic

WebSwags performs a recursive walk of the directory provided via `-root` (defaults to `..`). Any file ending in `.yaml`, `.yml`, or `.json` is considered a candidate spec.

- **Path Agnostic**: Specs can live anywhere (`apis/`, `docs/`, deeply nested folders, etc.).
- **Multi-Version Parsing**: Attempts OpenAPI 3.x first (via `kin-openapi`) and falls back to Swagger 2.0 (`go-openapi/spec`).
- **Smart Naming**: Service names derive from explicit titles, nearby folder names (e.g., before `spec/`, `api/`, `swagger/`), or ultimately the filename.
- **Metadata Extraction**: Captures title, version, description, format, and a served path for each spec.
- **Format Detection**: Chooses YAML vs JSON by extension with a content sniff fallback.

### API Endpoints

- `GET /` - Main service listing page with format indicators
- `GET /service/{service}` - Swagger UI for specific service (auto-detects format)
- `GET /api/specs` - JSON API listing all discovered specifications (includes format field)
- `GET /api/specs/{service}/swagger.yaml` - Raw YAML file for service
- `GET /api/specs/{service}/swagger.json` - Raw JSON file for service
- `GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD|CONNECT|TRACE /proxy?url={encoded-url}` - CORS proxy endpoint for API requests

### CORS Proxy

The built-in CORS proxy allows you to test API endpoints directly from Swagger UI without encountering CORS (Cross-Origin Resource Sharing) restrictions. This is especially useful when:

- Testing APIs hosted on different domains
- Working with APIs that don't have CORS headers configured
- Making requests to production/test servers from your local environment

**How it works:**

1. When you make a request from Swagger UI to an external API, the request is automatically intercepted
2. The request is proxied through the WebSwags server at `/proxy?url={target-url}`
3. The server makes the request on your behalf (server-to-server, no CORS restrictions)
4. The response is returned to your browser with appropriate CORS headers

**Usage:**

The proxy is transparent - just use the "Try it out" feature in Swagger UI normally. The application automatically routes external requests through the proxy.

Prefer to hit APIs directly? Each service page now includes a “Use Proxy” toggle that flips between proxied and direct requests and surfaces a colored status banner when you go direct.

You can also use the proxy directly:

```bash
# Direct proxy usage
curl "http://localhost:8085/proxy?url=https%3A%2F%2Fapi.example.com%2Fendpoint"
```

**Note:** The proxy adds `Access-Control-Allow-Origin: *` headers to all responses, allowing the Swagger UI to function properly.

### UI Controls

- **Theme Toggle**: The floating button (💻/☀️/🌙) cycles between system, light, and dark themes while persisting to `localStorage`.
- **Proxy Toggle**: Switch between proxied and direct API calls per service; the badge and banner make the current state obvious.
- **Viewer Toggle**: Instantly swap between Swagger UI and Redoc renders using the same discovered spec URL.
- **Format Badge**: Shows whether the source spec is YAML or JSON and adapts its color accordingly.

## Development

### Adding New Services

WebSwags automatically discovers new services when you drop OpenAPI specifications anywhere under your chosen root. A common convention is:

```text
apis/{service-name}/openapi.yaml
apis/{service-name}/openapi.json
connector/{service-name}/spec/{service-name}.yaml
connector/{service-name}/spec/{service-name}.json
```

Once the file exists on disk, restart (or hot-reload) the server and the new service appears automatically with derived naming, version, and format badges.

### Customization

To customize the appearance or functionality:

1. **Styling**: Edit the embedded assets inside `templates/` (`index-styles.css`, `service-styles.css`, `theme.css`, `SwaggerDark.css`).
2. **Discovery Logic**: Extend `discovery/discovery.go` if you need alternative heuristics or metadata.
3. **UI Layout & Behavior**: Tweak `templates/index.html`, `templates/service.html`, plus the helper scripts `service-script.js` (proxy/viewer toggles) and `theme.js` (theme management).
4. **Server Wiring**: Update `main.go` if you embed new templates or change routing.

### Dependencies

- **Go 1.25+**: Required for building and running
- **gorilla/mux**: HTTP router for handling requests
- **kin-openapi**: OpenAPI 3.x specification parsing
- **go-openapi/spec**: Swagger 2.0 specification parsing
- **sigs.k8s.io/yaml**: YAML processing utilities
- **golang.org/x/text**: Text processing and case conversion
- **Swagger UI**: Loaded via CDN (unpkg.com)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

### No Services Found

If no services are discovered:

1. Confirm spec files (`.yaml`, `.yml`, `.json`) exist somewhere under the configured `-root` path.
2. Ensure each file contains a valid `openapi:` or `swagger:` declaration near the top.
3. Verify the process has read permissions and the files aren’t hidden by `.gitignore` or tooling.
4. Check server logs for parsing errors; non-OpenAPI files are skipped at debug level.

### Service Not Loading

If a specific service won't load:

1. Verify the YAML file is valid OpenAPI/Swagger format
2. Confirm the `GET /api/specs` endpoint lists the service (case-sensitive names)
3. Look for parsing errors in server logs

### Port Already in Use

If port 8085 is already in use:

1. Stop other services using the port
2. Or modify the `port` constant in `main.go`
3. Update the Makefile accordingly
