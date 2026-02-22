(function () {
    var STORAGE_KEY = 'ask-theme';

    function getSystemTheme() {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function applyTheme(preference) {
        var resolved = preference === 'system' ? getSystemTheme() : preference;
        document.documentElement.setAttribute('data-theme', resolved);
        document.querySelectorAll('.theme-btn').forEach(function (btn) {
            btn.classList.toggle('active', btn.getAttribute('data-theme') === preference);
        });
    }

    function init() {
        var preference = localStorage.getItem(STORAGE_KEY) || 'system';
        applyTheme(preference);

        document.querySelectorAll('.theme-btn').forEach(function (btn) {
            btn.addEventListener('click', function () {
                var pref = btn.getAttribute('data-theme');
                localStorage.setItem(STORAGE_KEY, pref);
                applyTheme(pref);
            });
        });

        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function () {
            var pref = localStorage.getItem(STORAGE_KEY) || 'system';
            if (pref === 'system') applyTheme('system');
        });
    }

    // Immediate application (backup for inline script in <head>)
    var p = localStorage.getItem(STORAGE_KEY) || 'system';
    document.documentElement.setAttribute('data-theme', p === 'system' ? getSystemTheme() : p);

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Map tile helpers for Leaflet pages
    window.askTheme = {
        getTileUrl: function () {
            var theme = document.documentElement.getAttribute('data-theme');
            return theme === 'dark'
                ? 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png'
                : 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png';
        },
        getTileAttribution: function () {
            return '\u00a9 <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> \u00a9 <a href="https://carto.com/attributions">CARTO</a>';
        }
    };
})();
