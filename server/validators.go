package server

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	defaultSearchLimit = 100
	maxSearchLimit     = 500
)

var (
	icaoCodeRe    = regexp.MustCompile(`^[a-zA-Z]{4}$`)
	countryCodeRe = regexp.MustCompile(`^[a-zA-Z]{2}$`)
)

var validAirportTypes = map[string]bool{
	"large_airport":  true,
	"medium_airport": true,
	"small_airport":  true,
	"heliport":       true,
	"seaplane_base":  true,
	"closed":         true,
	"balloonport":    true,
}

// parseAirportTypes splits a comma-separated type string, trims whitespace,
// validates each type, and returns the list. Returns false if any type is invalid.
func parseAirportTypes(typesStr string) ([]string, bool) {
	if typesStr == "" {
		return nil, true
	}
	parts := strings.Split(typesStr, ",")
	types := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if !validAirportTypes[t] {
			return nil, false
		}
		types = append(types, t)
	}
	return types, true
}

// isValidSearchQuery accepts 2 to 100 printable characters in any script.
// Queries are bound as SQL parameters, so no character needs filtering for
// safety; the bounds only keep the scan meaningful and cheap.
func isValidSearchQuery(q string) bool {
	n := utf8.RuneCountInString(q)
	if n < 2 || n > 100 {
		return false
	}
	for _, r := range q {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// isValidCountryCode validates an ISO 3166-1 alpha-2 country code
func isValidCountryCode(code string) bool {
	return countryCodeRe.MatchString(code)
}

// isValidICAOCode validates that the ICAO code is exactly 4 letters
func isValidICAOCode(code string) bool {
	return icaoCodeRe.MatchString(code)
}

// parseSearchLimit returns the default limit for an empty string, and false
// when the limit is not an integer in [1, maxSearchLimit].
func parseSearchLimit(s string) (int, bool) {
	if s == "" {
		return defaultSearchLimit, true
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > maxSearchLimit {
		return 0, false
	}
	return n, true
}

// isValidRange validates that the range string is a positive number up to 10800 NM (half Earth circumference)
func isValidRange(rangeStr string) (float64, bool) {
	rangeNM, err := strconv.ParseFloat(rangeStr, 64)
	if err != nil {
		return 0, false
	}
	if rangeNM <= 0 || rangeNM > 10800 {
		return 0, false
	}
	return rangeNM, true
}
