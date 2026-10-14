// Shared helpers for every page: theme switching, API calls, airport
// formatting, code autocomplete and map tiles.
(function () {
    var STORAGE_KEY = 'ask-theme';
    var media = window.matchMedia('(prefers-color-scheme: dark)');

    function applyTheme(preference) {
        var resolved = preference === 'system' ? (media.matches ? 'dark' : 'light') : preference;
        document.documentElement.setAttribute('data-theme', resolved);
        document.querySelectorAll('.theme-btn').forEach(function (btn) {
            btn.setAttribute('aria-pressed', btn.dataset.theme === preference ? 'true' : 'false');
        });
        window.dispatchEvent(new CustomEvent('theme-changed', { detail: { theme: resolved } }));
    }

    function preference() {
        return localStorage.getItem(STORAGE_KEY) || 'system';
    }

    applyTheme(preference());
    document.querySelectorAll('.theme-btn').forEach(function (btn) {
        btn.addEventListener('click', function () {
            localStorage.setItem(STORAGE_KEY, btn.dataset.theme);
            applyTheme(btn.dataset.theme);
        });
    });
    media.addEventListener('change', function () {
        if (preference() === 'system') applyTheme('system');
    });

    // Navigation feedback. Pages render on the server, so a slow one (a cold
    // start, a busy machine) shows nothing until it arrives: say so on the
    // page being left. A bar starts at once; a spinner joins if the wait
    // lasts.
    var loader = null;
    var loaderTimer = null;

    function showLoading(label) {
        if (!loader) {
            loader = document.createElement('div');
            loader.className = 'nav-loading';
            loader.setAttribute('role', 'status');
            loader.innerHTML = '<div class="nav-loading-bar"></div>' +
                '<div class="nav-loading-panel"><span class="nav-loading-spinner" aria-hidden="true"></span>' +
                '<span class="nav-loading-label"></span></div>';
            document.body.appendChild(loader);
        }
        loader.querySelector('.nav-loading-label').textContent = label;
        loader.classList.add('active');
        clearTimeout(loaderTimer);
        loaderTimer = setTimeout(function () { loader.classList.add('slow'); }, 400);
    }

    function hideLoading() {
        clearTimeout(loaderTimer);
        if (loader) loader.classList.remove('active', 'slow');
    }

    document.addEventListener('click', function (e) {
        var a = e.target.closest('a[href]');
        if (!a || e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
        if (a.target && a.target !== '_self') return;
        var url = new URL(a.href, location.href);
        if (url.origin !== location.origin || a.hasAttribute('download')) return;
        // Same page, different hash: no navigation.
        if (url.pathname === location.pathname && url.search === location.search && url.hash) return;
        var m = /^\/airport\/([^/]+)$/.exec(url.pathname);
        showLoading(m ? 'Loading ' + decodeURIComponent(m[1]).toUpperCase() + '\u2026' : 'Loading\u2026');
    });

    // Forms that navigate (the home page search); pages that answer with
    // fetch prevent the default and keep their own status line.
    document.addEventListener('submit', function (e) {
        setTimeout(function () {
            if (!e.defaultPrevented) showLoading('Searching\u2026');
        }, 0);
    });

    // Back and forward restore pages from the cache, loader included.
    window.addEventListener('pageshow', hideLoading);

    var TYPE_LABELS = {
        large_airport: 'Large airport',
        medium_airport: 'Medium airport',
        small_airport: 'Small airport',
        heliport: 'Heliport',
        seaplane_base: 'Seaplane base',
        closed: 'Closed',
        balloonport: 'Balloonport'
    };

    function esc(text) {
        var div = document.createElement('div');
        div.textContent = text == null ? '' : String(text);
        return div.innerHTML;
    }

    // The 4-letter code the API can look this airport up by, or null.
    function lookupCode(ap) {
        var candidates = [ap.icao_code, ap.ident, ap.gps_code];
        for (var i = 0; i < candidates.length; i++) {
            if (/^[A-Z]{4}$/.test(candidates[i] || '')) return candidates[i];
        }
        return null;
    }

    window.ask = {
        esc: esc,
        lookupCode: lookupCode,

        typeLabel: function (type) {
            return TYPE_LABELS[type] || type || '';
        },

        // ICAO codes are drawn as airfield location signs; airports without
        // one show their OurAirports ident unsigned.
        sign: function (ap, extraClass) {
            var cls = 'sign' + (extraClass ? ' ' + extraClass : '');
            if (ap.icao_code) return '<span class="' + cls + '">' + esc(ap.icao_code) + '</span>';
            return '<span class="' + cls + ' sign-none" title="No ICAO code">' + esc(ap.ident) + '</span>';
        },

        // The airport page: by ICAO code when there is one, else by ident.
        airportURL: function (ap) {
            return '/airport/' + encodeURIComponent(lookupCode(ap) || ap.ident);
        },

        place: function (ap) {
            return [ap.municipality, ap.iso_country].filter(Boolean).map(esc).join(', ');
        },

        fetchJSON: async function (url) {
            var response = await fetch(url);
            if (!response.ok) {
                var text = (await response.text()).trim();
                throw new Error(text || 'The server answered ' + response.status + '.');
            }
            return response.json();
        },

        setStatus: function (el, message, isError) {
            el.textContent = message || '';
            el.classList.toggle('error', !!isError);
        },

        // Uppercase a 4-letter code input; anything else is left alone.
        normalizeCode: function (value) {
            value = value.trim();
            return /^[a-zA-Z]{4}$/.test(value) ? value.toUpperCase() : value;
        },

        // Suggest airports in a <datalist> as the user types a name or code.
        // Options carry the 4-letter code, so picking one fills the field.
        autocomplete: function (input) {
            var list = document.getElementById(input.getAttribute('list'));
            var timer = null;
            var last = '';
            input.addEventListener('input', function () {
                clearTimeout(timer);
                var q = input.value.trim();
                if (q.length < 2 || q === last || /^[A-Z]{4}$/.test(q)) return;
                timer = setTimeout(async function () {
                    last = q;
                    try {
                        var data = await window.ask.fetchJSON('/api/airport/search?limit=8&name=' + encodeURIComponent(q));
                        list.innerHTML = data.airports
                            .filter(lookupCode)
                            .map(function (ap) {
                                return '<option value="' + esc(lookupCode(ap)) + '">' + esc(ap.name) +
                                    (ap.municipality ? ', ' + esc(ap.municipality) : '') + '</option>';
                            })
                            .join('');
                    } catch (e) {
                        list.innerHTML = '';
                    }
                }, 200);
            });
        },

        // Local time at an airport, as "14:03 UTC+02:00 (Europe/Paris)".
        localTime: async function (code) {
            var t = await window.ask.fetchJSON('/api/airport/time?icao=' + encodeURIComponent(code));
            return t.local_time.slice(11, 16) + ' UTC' + t.utc_offset + ' (' + t.timezone + ')';
        },

        // Add the basemap: the self-hosted Protomaps vector tiles when the
        // server has them (body[data-tiles]), else OpenStreetMap's raster
        // tiles, which the dark theme inverts in CSS.
        addTiles: function (map) {
            var url = document.body.dataset.tiles;
            if (url && window.protomapsL) {
                map.getContainer().classList.add('vector-tiles');
                var layer = null;
                var attribution = '<a href="https://protomaps.com">Protomaps</a> &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';
                var draw = function () {
                    if (layer) map.removeLayer(layer);
                    var dark = document.documentElement.getAttribute('data-theme') === 'dark';
                    layer = protomapsL.leafletLayer({ url: url, flavor: dark ? 'dark' : 'light', lang: 'en', attribution: attribution });
                    layer.addTo(map);
                    layer.bringToBack();
                };
                draw();
                window.addEventListener('theme-changed', draw);
                map.on('unload', function () { window.removeEventListener('theme-changed', draw); });
                return;
            }
            L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
                maxZoom: 19
            }).addTo(map);
        },

        // Read the CSS custom property, so map overlays follow the theme.
        cssVar: function (name) {
            return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
        }
    };
})();
