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

        // Add OpenStreetMap tiles. The dark theme inverts them in CSS, so
        // the theme can change without reloading tiles.
        addTiles: function (map) {
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
