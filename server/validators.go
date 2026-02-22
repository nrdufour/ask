package server

import (
	"regexp"
	"strconv"
)

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
