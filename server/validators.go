package server

import (
	"regexp"
	"strconv"
	"strings"
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

// isValidAirportType checks if the given type string is a known airport type
func isValidAirportType(t string) bool {
	return validAirportTypes[t]
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
		if !isValidAirportType(t) {
			return nil, false
		}
		types = append(types, t)
	}
	return types, true
}

// isValidSearchParameter validates that the search parameter contains only allowed characters
func isValidSearchParameter(param string) bool {
	// Allow letters, spaces, hyphens, apostrophes, and common punctuation for airport names
	matched, _ := regexp.MatchString(`^[a-zA-Z\s\-'\.]+$`, param)
	return matched
}

// isValidCountryCode validates that the country code contains only letters
func isValidCountryCode(code string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z]+$`, code)
	return matched
}

// isValidICAOCode validates that the ICAO code is exactly 4 letters
func isValidICAOCode(code string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z]{4}$`, code)
	return matched
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
