package server

import "regexp"

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
