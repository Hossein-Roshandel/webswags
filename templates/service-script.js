// Proxy toggle state management
const PROXY_STORAGE_KEY = 'webswags-proxy-enabled';
const VIEWER_STORAGE_KEY = 'webswags-viewer-mode';
let proxyEnabled = localStorage.getItem(PROXY_STORAGE_KEY) !== 'false'; // default to true
let viewerMode = localStorage.getItem(VIEWER_STORAGE_KEY) || 'swagger'; // 'swagger' or 'redoc'

// Update UI based on proxy state
function updateProxyUI() {
    const toggle = document.getElementById('proxyToggle');
    const status = document.getElementById('proxyStatus');
    const corsInfo = document.getElementById('corsInfo');

    if (proxyEnabled) {
        toggle.classList.add('active');
        status.textContent = 'ON';
        status.classList.remove('direct');
        corsInfo.innerHTML = '<strong>🔓 CORS Proxy Enabled</strong>API requests are automatically proxied to avoid CORS issues.';
        corsInfo.style.background = '#3498db';
    } else {
        toggle.classList.remove('active');
        status.textContent = 'OFF';
        status.classList.add('direct');
        corsInfo.innerHTML = '<strong>⚠️ Direct Mode</strong>Requests sent directly to API servers. CORS errors may occur.';
        corsInfo.style.background = '#e74c3c';
    }
}

// Toggle proxy state
document.getElementById('proxyToggle').addEventListener('click', function () {
    proxyEnabled = !proxyEnabled;
    localStorage.setItem(PROXY_STORAGE_KEY, proxyEnabled);
    updateProxyUI();

    // Show reload prompt
    if (confirm('Proxy mode changed. Reload page to apply changes?')) {
        window.location.reload();
    }
});

// Toggle viewer mode
function toggleViewer() {
    viewerMode = viewerMode === 'swagger' ? 'redoc' : 'swagger';
    localStorage.setItem(VIEWER_STORAGE_KEY, viewerMode);
    window.location.reload();
}

// Initialize UI
updateProxyUI();

// Set format badge color
const formatBadge = document.querySelector('.format-badge');
const format = formatBadge.dataset.format;
if (format === 'json') {
    formatBadge.style.background = '#f39c12'; // Orange for JSON
} else {
    formatBadge.style.background = '#27ae60'; // Green for YAML
}

// Update viewer toggle button
const viewerToggle = document.getElementById('viewerToggle');
if (viewerToggle) {
    viewerToggle.textContent = viewerMode === 'swagger' ? '📖 Switch to Redoc' : '🔧 Switch to Swagger';
    viewerToggle.addEventListener('click', toggleViewer);
}

// Initialize appropriate viewer
window.onload = function () {
    if (viewerMode === 'redoc') {
        initializeRedoc();
    } else {
        initializeSwagger();
    }
};

function initializeSwagger() {
    const container = document.getElementById('swagger-ui');
    container.style.display = 'block';

    const ui = SwaggerUIBundle({
        url: swaggerSpecURL,
        dom_id: '#swagger-ui',
        deepLinking: true,
        displayOperationId: false,
        defaultModelsExpandDepth: 1,
        defaultModelExpandDepth: 1,
        docExpansion: 'list', // 'list', 'full', or 'none'
        filter: true, // Enable search/filter box
        showExtensions: true,
        showCommonExtensions: true,
        persistAuthorization: true, // Keep auth tokens on page refresh
        tryItOutEnabled: true,
        supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch', 'head', 'options'],
        presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIStandalonePreset
        ],
        plugins: [
            SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout",
        requestInterceptor: function (req) {
            // Only proxy if enabled and it's an external request
            if (proxyEnabled &&
                !req.url.startsWith(window.location.origin) &&
                !req.url.startsWith('/') &&
                !req.url.includes('/proxy')) {
                console.log('Proxying request to:', req.url);
                req.url = '/proxy?url=' + encodeURIComponent(req.url);
            } else if (!proxyEnabled) {
                console.log('Direct request to:', req.url);
            }
            return req;
        }
    });

    // Enable syntax highlighting for responses
    ui.initOAuth({
        clientId: "your-client-id",
        clientSecret: "your-client-secret-if-required",
        realm: "your-realms",
        appName: "WebSwags",
        scopeSeparator: " ",
        scopes: "openid profile email",
        additionalQueryStringParams: {},
        useBasicAuthenticationWithAccessCodeGrant: false,
        usePkceWithAuthorizationCodeGrant: false
    });
}

function initializeRedoc() {
    const container = document.getElementById('swagger-ui');
    container.style.display = 'block';
    container.innerHTML = '<div id="redoc-container"></div>';

    // Get current theme
    const root = document.documentElement;
    const currentTheme = root.getAttribute('data-theme') || 'light';
    const isDark = currentTheme === 'dark';

    // Redoc doesn't support request interceptor, so we need to handle proxy differently
    // For now, we'll just pass the spec URL directly
    // If proxy is needed, the server should handle it at the spec URL level

    Redoc.init(
        swaggerSpecURL,
        {
            scrollYOffset: 50,
            hideDownloadButton: false,
            disableSearch: false,
            expandResponses: '200,201',
            jsonSampleExpandLevel: 2,
            hideSingleRequestSampleTab: true,
            menuToggle: true,
            nativeScrollbars: false,
            noAutoAuth: false,
            pathInMiddlePanel: false,
            requiredPropsFirst: true,
            sortPropsAlphabetically: true,
            suppressWarnings: false,
            payloadSampleIdx: 0,
            theme: {
                colors: {
                    primary: {
                        main: '#667eea'
                    }
                },
                typography: {
                    fontSize: '14px',
                    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                    headings: {
                        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
                    }
                },
                sidebar: {
                    backgroundColor: isDark ? '#2d2d2d' : '#fafafa',
                    textColor: isDark ? '#e0e0e0' : '#333'
                },
                rightPanel: {
                    backgroundColor: isDark ? '#1a1a1a' : '#263238',
                    textColor: isDark ? '#e0e0e0' : '#ffffff'
                }
            }
        },
        document.getElementById('redoc-container')
    );
}
