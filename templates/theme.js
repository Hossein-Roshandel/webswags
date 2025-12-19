// Theme management
const THEME_STORAGE_KEY = 'webswags-theme';
let currentTheme = localStorage.getItem(THEME_STORAGE_KEY) || 'system'; // 'light', 'dark', or 'system'

// SwaggerDark CSS content
const swaggerDarkCSS = `{{template "SwaggerDark.css"}}`;

// Apply theme based on preference
function applyTheme(theme) {
    const root = document.documentElement;

    let effectiveTheme;
    if (theme === 'system') {
        // Check system preference
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        effectiveTheme = systemPrefersDark ? 'dark' : 'light';
    } else {
        effectiveTheme = theme;
    }

    root.setAttribute('data-theme', effectiveTheme);

    // Apply Swagger dark theme if needed
    applySwaggerDarkTheme(effectiveTheme);
}

// Apply or remove SwaggerDark.css for Swagger UI
function applySwaggerDarkTheme(effectiveTheme) {
    const styleElement = document.getElementById('swagger-dark-styles');
    if (!styleElement) return; // Not on a page with Swagger UI

    if (effectiveTheme === 'dark') {
        styleElement.textContent = swaggerDarkCSS;
    } else {
        styleElement.textContent = '';
    }
}

// Update theme toggle UI
function updateThemeUI() {
    const themeToggle = document.getElementById('themeToggle');
    if (!themeToggle) return;

    const icons = {
        light: '☀️',
        dark: '🌙',
        system: '💻'
    };

    const labels = {
        light: 'Light',
        dark: 'Dark',
        system: 'System'
    };

    themeToggle.innerHTML = `${icons[currentTheme]} ${labels[currentTheme]}`;
}

// Cycle through themes
function cycleTheme() {
    const themes = ['light', 'dark', 'system'];
    const currentIndex = themes.indexOf(currentTheme);
    currentTheme = themes[(currentIndex + 1) % themes.length];

    localStorage.setItem(THEME_STORAGE_KEY, currentTheme);
    applyTheme(currentTheme);
    updateThemeUI();
}

// Initialize theme
applyTheme(currentTheme);
updateThemeUI();

// Listen for system theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    if (currentTheme === 'system') {
        applyTheme('system');
    }
});

// Setup theme toggle button
const themeToggle = document.getElementById('themeToggle');
if (themeToggle) {
    themeToggle.addEventListener('click', cycleTheme);
}
