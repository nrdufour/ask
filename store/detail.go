package store

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// NavaidRadiusNM is how far from an airport Detail looks for navaids.
const NavaidRadiusNM = 30

// Where a runway end's heading comes from, most accurate first.
const (
	HeadingFromData       = "data"       // le/he_heading_degT, true
	HeadingFromThresholds = "thresholds" // computed from both threshold positions, true
	HeadingFromDesignator = "designator" // "08L" -> 080, magnetic and rounded
)

type RunwayEnd struct {
	Ident        string   `json:"ident"`
	LatitudeDeg  *float64 `json:"latitude_deg"`
	LongitudeDeg *float64 `json:"longitude_deg"`
	ElevationFt  *int     `json:"elevation_ft"`
	// HeadingDeg is the runway's direction when landing on this end. It is
	// true unless HeadingSource is "designator"; nil when unknown.
	HeadingDeg           *float64 `json:"heading_deg"`
	HeadingSource        string   `json:"heading_source,omitempty"`
	DisplacedThresholdFt *int     `json:"displaced_threshold_ft"`
}

type Runway struct {
	LengthFt *int   `json:"length_ft"`
	WidthFt  *int   `json:"width_ft"`
	Surface  string `json:"surface"`
	// SurfaceLabel reads the free-form surface code: "ASPH-G" -> "asphalt".
	SurfaceLabel string    `json:"surface_label"`
	Lighted      bool      `json:"lighted"`
	Closed       bool      `json:"closed"`
	LowEnd       RunwayEnd `json:"le"`
	HighEnd      RunwayEnd `json:"he"`
}

type Frequency struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	MHz         float64 `json:"frequency_mhz"`
}

type Navaid struct {
	Ident        string  `json:"ident"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	FrequencyKHz *int    `json:"frequency_khz"`
	DMEChannel   string  `json:"dme_channel"`
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
	// DistanceNM and BearingDeg (true) locate the navaid from the airport.
	DistanceNM float64 `json:"distance_nm"`
	BearingDeg float64 `json:"bearing_deg"`
}

type AirportDetail struct {
	Airport     Airport     `json:"airport"`
	CountryName string      `json:"country_name"`
	RegionName  string      `json:"region_name"`
	Runways     []Runway    `json:"runways"`
	Frequencies []Frequency `json:"frequencies"`
	Navaids     []Navaid    `json:"navaids"`
}

// HasRunwayHeadings reports whether any open runway end has a known heading,
// which wind components need.
func (d *AirportDetail) HasRunwayHeadings() bool {
	for _, r := range d.Runways {
		if !r.Closed && (r.LowEnd.HeadingDeg != nil || r.HighEnd.HeadingDeg != nil) {
			return true
		}
	}
	return false
}

var identRe = regexp.MustCompile(`^[A-Za-z0-9-]{2,12}$`)

// Detail returns an airport with its runways, frequencies and nearby navaids.
// code is a 4-letter code as ByCode takes, or any OurAirports ident
// ("FR-0496"), so airports without an ICAO code have a page too.
func (s *Store) Detail(code string) (*AirportDetail, error) {
	a, err := s.resolve(code)
	if err != nil {
		return nil, err
	}

	d := &AirportDetail{Airport: *a}
	var country, region sql.NullString
	err = s.db.QueryRow(`SELECT
		(SELECT name FROM countries WHERE code = ?1),
		(SELECT name FROM regions WHERE code = ?2)`, a.IsoCountry, a.IsoRegion).Scan(&country, &region)
	if err != nil {
		return nil, err
	}
	d.CountryName, d.RegionName = country.String, region.String

	if d.Runways, err = s.runways(a.ID); err != nil {
		return nil, fmt.Errorf("runways: %w", err)
	}
	if d.Frequencies, err = s.frequencies(a.ID); err != nil {
		return nil, fmt.Errorf("frequencies: %w", err)
	}
	if d.Navaids, err = s.navaidsNear(a.LatitudeDeg, a.LongitudeDeg, NavaidRadiusNM); err != nil {
		return nil, fmt.Errorf("navaids: %w", err)
	}
	return d, nil
}

func (s *Store) resolve(code string) (*Airport, error) {
	if code == "" {
		return nil, inputErrorf("an airport code is required")
	}
	if icaoCodeRe.MatchString(code) {
		return s.ByCode(code)
	}
	if !identRe.MatchString(code) {
		return nil, inputErrorf("%q is not an airport code", code)
	}
	ident := strings.ToUpper(code)
	a, err := scanAirport(s.db.QueryRow(`SELECT `+airportColumns+` FROM airports WHERE ident = ?`, ident))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, ident)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func floatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	return &n.Float64
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	i := int(n.Int64)
	return &i
}

func (s *Store) runways(airportID int) ([]Runway, error) {
	rows, err := s.db.Query(`SELECT length_ft, width_ft, surface, lighted, closed,
		le_ident, le_latitude_deg, le_longitude_deg, le_elevation_ft, le_heading_degT, le_displaced_threshold_ft,
		he_ident, he_latitude_deg, he_longitude_deg, he_elevation_ft, he_heading_degT, he_displaced_threshold_ft
		FROM runways WHERE airport_ref = ?
		ORDER BY closed, length_ft DESC`, airportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runways := []Runway{}
	for rows.Next() {
		var (
			length, width, lighted, closed           sql.NullInt64
			surface, leIdent, heIdent                sql.NullString
			leLat, leLon, leHdg, heLat, heLon, heHdg sql.NullFloat64
			leElev, leDisp, heElev, heDisp           sql.NullInt64
		)
		if err := rows.Scan(&length, &width, &surface, &lighted, &closed,
			&leIdent, &leLat, &leLon, &leElev, &leHdg, &leDisp,
			&heIdent, &heLat, &heLon, &heElev, &heHdg, &heDisp); err != nil {
			return nil, err
		}

		r := Runway{
			LengthFt:     intPtr(length),
			WidthFt:      intPtr(width),
			Surface:      surface.String,
			SurfaceLabel: surfaceLabel(surface.String),
			Lighted:      lighted.Int64 == 1,
			Closed:       closed.Int64 == 1,
			LowEnd: RunwayEnd{
				Ident: leIdent.String, LatitudeDeg: floatPtr(leLat), LongitudeDeg: floatPtr(leLon),
				ElevationFt: intPtr(leElev), HeadingDeg: floatPtr(leHdg), DisplacedThresholdFt: intPtr(leDisp),
			},
			HighEnd: RunwayEnd{
				Ident: heIdent.String, LatitudeDeg: floatPtr(heLat), LongitudeDeg: floatPtr(heLon),
				ElevationFt: intPtr(heElev), HeadingDeg: floatPtr(heHdg), DisplacedThresholdFt: intPtr(heDisp),
			},
		}
		resolveHeadings(&r)
		runways = append(runways, r)
	}
	return runways, rows.Err()
}

// resolveHeadings fills in missing runway headings: from the two threshold
// positions when the data has them, else from the designator.
func resolveHeadings(r *Runway) {
	le, he := &r.LowEnd, &r.HighEnd
	for _, end := range []*RunwayEnd{le, he} {
		if end.HeadingDeg != nil {
			end.HeadingSource = HeadingFromData
		}
	}

	if le.LatitudeDeg != nil && le.LongitudeDeg != nil && he.LatitudeDeg != nil && he.LongitudeDeg != nil {
		if le.HeadingDeg == nil {
			h := roundTo(initialBearing(*le.LatitudeDeg, *le.LongitudeDeg, *he.LatitudeDeg, *he.LongitudeDeg), 1)
			le.HeadingDeg, le.HeadingSource = &h, HeadingFromThresholds
		}
		if he.HeadingDeg == nil {
			h := roundTo(initialBearing(*he.LatitudeDeg, *he.LongitudeDeg, *le.LatitudeDeg, *le.LongitudeDeg), 1)
			he.HeadingDeg, he.HeadingSource = &h, HeadingFromThresholds
		}
	}

	for _, end := range []*RunwayEnd{le, he} {
		if end.HeadingDeg == nil {
			if h, ok := designatorHeading(end.Ident); ok {
				end.HeadingDeg, end.HeadingSource = &h, HeadingFromDesignator
			}
		}
	}
}

// designatorHeading reads the magnetic heading in a runway designator:
// "08L" -> 80, "36" -> 360. Designators without a number ("N", "H1") have none.
func designatorHeading(ident string) (float64, bool) {
	digits := strings.TrimRight(ident, "LCRGWUS")
	n, err := strconv.Atoi(digits)
	if err != nil || len(digits) > 2 || n < 1 || n > 36 {
		return 0, false
	}
	return float64(n * 10), true
}

// surfaceLabel reads the free-form surface codes of the dataset ("ASP",
// "ASPH-G", "CONC", "GRS", "TURF-F", "Grass / Sod") as a plain material.
func surfaceLabel(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	prefixes := []struct{ prefix, label string }{
		{"ASP", "asphalt"}, {"BIT", "asphalt"}, {"PEM", "asphalt"}, {"TAR", "asphalt"},
		{"CON", "concrete"}, {"PSP", "steel planking"}, {"MAT", "steel planking"},
		{"TURF", "grass"}, {"GRS", "grass"}, {"GRASS", "grass"}, {"SOD", "grass"},
		{"GRE", "gravel"}, {"GRV", "gravel"}, {"GVL", "gravel"}, {"GRAVEL", "gravel"},
		{"DIRT", "dirt"}, {"EARTH", "dirt"}, {"CLA", "clay"}, {"SAN", "sand"}, {"COR", "coral"},
		{"WATER", "water"}, {"SNOW", "snow"}, {"ICE", "ice"},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(c, p.prefix) {
			return p.label
		}
	}
	return strings.ToLower(code)
}

func (s *Store) frequencies(airportID int) ([]Frequency, error) {
	rows, err := s.db.Query(`SELECT type, description, frequency_mhz FROM frequencies
		WHERE airport_ref = ? AND frequency_mhz IS NOT NULL
		ORDER BY CASE type
			WHEN 'ATIS' THEN 0 WHEN 'CLD' THEN 1 WHEN 'DEL' THEN 1 WHEN 'GND' THEN 2
			WHEN 'TWR' THEN 3 WHEN 'CTAF' THEN 3 WHEN 'UNIC' THEN 4 WHEN 'AFIS' THEN 4
			WHEN 'APP' THEN 5 WHEN 'DEP' THEN 6 ELSE 9 END, frequency_mhz`, airportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	freqs := []Frequency{}
	for rows.Next() {
		var typ, desc sql.NullString
		var mhz float64
		if err := rows.Scan(&typ, &desc, &mhz); err != nil {
			return nil, err
		}
		freqs = append(freqs, Frequency{Type: typ.String, Description: desc.String, MHz: mhz})
	}
	return freqs, rows.Err()
}

func (s *Store) navaidsNear(lat, lon, radiusNM float64) ([]Navaid, error) {
	box := boundingBoxAround(lat, lon, radiusNM)
	query := `SELECT ident, name, type, frequency_khz, dme_channel, latitude_deg, longitude_deg
		FROM navaids WHERE latitude_deg BETWEEN ? AND ?`
	args := []any{box.MinLat, box.MaxLat}
	// Over the antimeridian or near a pole, filter by distance alone.
	if !box.AllLon && box.MinLon >= -180 && box.MaxLon <= 180 {
		query += ` AND longitude_deg BETWEEN ? AND ?`
		args = append(args, box.MinLon, box.MaxLon)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	navaids := []Navaid{}
	for rows.Next() {
		var ident, name, typ, channel sql.NullString
		var freq sql.NullInt64
		var nLat, nLon float64
		if err := rows.Scan(&ident, &name, &typ, &freq, &channel, &nLat, &nLon); err != nil {
			return nil, err
		}
		dist := calculateDistance(lat, lon, nLat, nLon)
		if dist > radiusNM {
			continue
		}
		bearing := 0.0
		if dist > 0.05 {
			bearing = roundTo(initialBearing(lat, lon, nLat, nLon), 0)
		}
		navaids = append(navaids, Navaid{
			Ident: ident.String, Name: name.String, Type: typ.String,
			FrequencyKHz: intPtr(freq), DMEChannel: channel.String,
			LatitudeDeg: nLat, LongitudeDeg: nLon,
			DistanceNM: roundTo(dist, 1), BearingDeg: math.Mod(bearing, 360),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(navaids, func(i, j int) bool { return navaids[i].DistanceNM < navaids[j].DistanceNM })
	return navaids, nil
}
