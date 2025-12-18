# WebSwags - API Documentation Server

WebSwags is a lightweight Go web server that provides a modern web interface for viewing OpenAPI/Swagger specifications in your projects.

## Features

- 🔍 **Auto-Discovery**: Automatically discovers OpenAPI/Swagger files in `connector/*/spec/` directories
- 📁 **Dual Format Support**: Supports both YAML (`.yaml`, `.yml`) and JSON (`.json`) specifications
- 🌐 **Modern UI**: Clean, responsive interface using Swagger UI 5.x
- 🏷️ **Format Indicators**: Visual badges showing file format (YAML/JSON) for each service
- 📋 **Service Listing**: Overview page showing all available API services with format information
- 🚀 **Quick Access**: Direct links to each service's documentation
- 🔓 **CORS Proxy**: Built-in proxy server to bypass CORS restrictions when testing API endpoints
- 🎯 **Development Focus**: Designed specifically for local development workflow

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
│   ├── index.html      # Main service listing page template
│   └── service.html    # Individual service Swagger UI template
└── README.md           # This documentation
```

### Discovery Logic

The server automatically scans for OpenAPI specifications:

- **Path Pattern**: `{root}/connector/*/spec/*.{yaml,yml,json}` (root configurable via `-root` flag)
- **Supported Formats**: OpenAPI 3.x and Swagger 2.0 in both YAML and JSON
- **Format Detection**: Automatic format detection from file extension
- **Extraction**: Automatically extracts title, version, and description from both YAML and JSON files

### API Endpoints

- `GET /` - Main service listing page with format indicators
- `GET /service/{service}` - Swagger UI for specific service (auto-detects format)
- `GET /api/specs` - JSON API listing all discovered specifications (includes format field)
- `GET /api/specs/{service}/swagger.yaml` - Raw YAML file for service
- `GET /api/specs/{service}/swagger.json` - Raw JSON file for service
- `GET|POST|PUT|PATCH|DELETE /proxy?url={encoded-url}` - CORS proxy endpoint for API requests

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

You can also use the proxy directly:

```bash
# Direct proxy usage
curl "http://localhost:8085/proxy?url=https%3A%2F%2Fapi.example.com%2Fendpoint"
```

**Note:** The proxy adds `Access-Control-Allow-Origin: *` headers to all responses, allowing the Swagger UI to function properly.

## Development

### Adding New Services

WebSwags automatically discovers new services when you add OpenAPI specifications to:

```
connector/{service-name}/spec/{service-name}.yaml
connector/{service-name}/spec/{service-name}.json
```

The service will appear in the listing on the next server restart with the appropriate format badge.

### Customization

To customize the appearance or functionality:

1. **Styling**: Modify the CSS in the `handleIndex` function
2. **Discovery Logic**: Update the `discoverSwaggerSpecs` function
3. **UI Layout**: Modify the HTML templates in the handlers

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

1. Check that YAML files exist in `connector/*/spec/` directories
2. Ensure YAML files contain valid `openapi:` or `swagger:` declarations
3. Verify file permissions allow reading
4. Check server logs for parsing errors

### Service Not Loading

If a specific service won't load:

1. Verify the YAML file is valid OpenAPI/Swagger format
2. Check the file path matches the pattern `connector/{service}/spec/{service}.yaml`
3. Look for parsing errors in server logs

### Port Already in Use

If port 8085 is already in use:

1. Stop other services using the port
2. Or modify the `port` constant in `main.go`
3. Update the Makefile accordingly
