package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const airportColumns = `id, ident, type, name, latitude_deg, longitude_deg, elevation_ft,
	continent, iso_country, iso_region, municipality, scheduled_service,
	icao_code, iata_code, gps_code, local_code, home_link, wikipedia_link, keywords`

// typeRank sorts airports by relevance: big airports first, closed ones last.
const typeRank = `CASE type
	WHEN 'large_airport' THEN 0
	WHEN 'medium_airport' THEN 1
	WHEN 'small_airport' THEN 2
	WHEN 'closed' THEN 9
	ELSE 5 END`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAirport(row rowScanner) (Airport, error) {
	var (
		id                                                 sql.NullInt64
		lat, lon                                           sql.NullFloat64
		elevation                                          sql.NullInt64
		ident, typ, name, continent, isoCountry, isoRegion sql.NullString
		municipality, scheduled, icao, iata, gps, local    sql.NullString
		homeLink, wikipediaLink, keywords                  sql.NullString
	)

	err := row.Scan(&id, &ident, &typ, &name, &lat, &lon, &elevation,
		&continent, &isoCountry, &isoRegion, &municipality, &scheduled,
		&icao, &iata, &gps, &local, &homeLink, &wikipediaLink, &keywords)
	if err != nil {
		return Airport{}, err
	}

	a := Airport{
		ID:               int(id.Int64),
		Ident:            ident.String,
		Type:             typ.String,
		Name:             name.String,
		LatitudeDeg:      lat.Float64,
		LongitudeDeg:     lon.Float64,
		Continent:        continent.String,
		IsoCountry:       isoCountry.String,
		IsoRegion:        isoRegion.String,
		Municipality:     municipality.String,
		ScheduledService: scheduled.String,
		IcaoCode:         icao.String,
		IataCode:         iata.String,
		GpsCode:          gps.String,
		LocalCode:        local.String,
		HomeLink:         homeLink.String,
		WikipediaLink:    wikipediaLink.String,
		Keywords:         keywords.String,
	}
	if elevation.Valid {
		e := int(elevation.Int64)
		a.ElevationFt = &e
	}
	return a, nil
}

func scanAirports(rows *sql.Rows) ([]Airport, error) {
	defer rows.Close()
	airports := []Airport{}
	for rows.Next() {
		a, err := scanAirport(rows)
		if err != nil {
			return nil, err
		}
		airports = append(airports, a)
	}
	return airports, rows.Err()
}

// ByCode looks an airport up by its 4-letter code. Most airports in the
// dataset have no icao_code, only an ident or GPS code that is the ICAO code
// in practice (FVHA, Harare), so those are tried next. Ties go to the most
// significant airport, which skips "closed" duplicates.
func (s *Store) ByCode(code string) (*Airport, error) {
	if err := checkCode(code); err != nil {
		return nil, err
	}
	code = strings.ToUpper(code)
	row := s.db.QueryRow(`SELECT `+airportColumns+` FROM airports
		WHERE icao_code = ?1 OR ident = ?1 OR gps_code = ?1
		ORDER BY CASE WHEN icao_code = ?1 THEN 0 WHEN ident = ?1 THEN 1 ELSE 2 END, `+typeRank+`
		LIMIT 1`, code)

	a, err := scanAirport(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, code)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Search finds airports whose name, city or keywords contain query, or whose
// code is exactly query, optionally in one country. Exact code matches come
// first. At most limit airports are returned.
func (s *Store) Search(query, country string, limit int) (*SearchResult, error) {
	query = strings.TrimSpace(query)
	if err := checkQuery(query); err != nil {
		return nil, err
	}
	if err := checkCountry(country); err != nil {
		return nil, err
	}
	if err := checkLimit(limit); err != nil {
		return nil, err
	}

	escaper := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	pattern := "%" + escaper.Replace(query) + "%"
	code := strings.ToUpper(query)

	codeMatch := `(icao_code = ?2 OR iata_code = ?2 OR ident = ?2 OR gps_code = ?2)`
	var countryArg any
	if country != "" {
		countryArg = strings.ToUpper(country)
	}

	sqlQuery := `SELECT ` + airportColumns + ` FROM airports
		WHERE (name LIKE ?1 ESCAPE '\' OR municipality LIKE ?1 ESCAPE '\' OR keywords LIKE ?1 ESCAPE '\' OR ` + codeMatch + `)
		AND (?3 IS NULL OR iso_country = ?3)
		ORDER BY CASE WHEN ` + codeMatch + ` THEN 0 ELSE 1 END, ` + typeRank + `, name
		LIMIT ?4`

	rows, err := s.db.Query(sqlQuery, pattern, code, countryArg, limit+1)
	if err != nil {
		return nil, err
	}
	airports, err := scanAirports(rows)
	if err != nil {
		return nil, err
	}

	result := &SearchResult{Airports: airports}
	if len(airports) > limit {
		result.Airports = airports[:limit]
		result.Truncated = true
	}
	result.Count = len(result.Airports)
	return result, nil
}

// Route returns the great-circle route between two airports.
func (s *Store) Route(from, to string) (*Route, error) {
	dep, err := s.ByCode(from)
	if err != nil {
		return nil, err
	}
	dest, err := s.ByCode(to)
	if err != nil {
		return nil, err
	}

	nm := calculateDistance(dep.LatitudeDeg, dep.LongitudeDeg, dest.LatitudeDeg, dest.LongitudeDeg)
	bearing := initialBearing(dep.LatitudeDeg, dep.LongitudeDeg, dest.LatitudeDeg, dest.LongitudeDeg)
	return &Route{
		DepartureAirport:   *dep,
		DestinationAirport: *dest,
		DistanceNM:         nm,
		DistanceKM:         nm * nmToKM,
		DistanceSM:         nm * nmToSM,
		InitialBearingDeg:  roundTo(bearing, 1),
	}, nil
}

// InRange finds the airports within rangeNM of the origin, nearest first.
// If types is non-empty, only airports of those types are returned.
func (s *Store) InRange(origin string, rangeNM float64, types []string) (*InRangeResult, error) {
	if err := checkRange(rangeNM); err != nil {
		return nil, err
	}
	if err := checkTypes(types); err != nil {
		return nil, err
	}
	o, err := s.ByCode(origin)
	if err != nil {
		return nil, err
	}
	airports, err := s.airportsInRange(o, rangeNM, types)
	if err != nil {
		return nil, err
	}
	return &InRangeResult{
		OriginAirport: *o,
		RangeNM:       rangeNM,
		Airports:      airports,
		Count:         len(airports),
	}, nil
}

// LocalTime returns the timezone and current local time at an airport.
func (s *Store) LocalTime(code string) (*AirportTime, error) {
	a, err := s.ByCode(code)
	if err != nil {
		return nil, err
	}
	finder, err := s.tzFinder()
	if err != nil {
		return nil, fmt.Errorf("loading the timezone finder: %w", err)
	}
	zone := finder.GetTimezoneName(a.LongitudeDeg, a.LatitudeDeg)
	if zone == "" {
		return nil, fmt.Errorf("no timezone found for %s", a.Code())
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, fmt.Errorf("loading timezone %s: %w", zone, err)
	}

	now := time.Now().In(loc)
	return &AirportTime{
		ICAO:      a.IcaoCode,
		Name:      a.Name,
		Timezone:  zone,
		LocalTime: now.Format(time.RFC3339),
		UTCOffset: now.Format("-07:00"),
	}, nil
}

// Countries lists every country, by name.
func (s *Store) Countries() ([]Country, error) {
	rows, err := s.db.Query(`SELECT id, code, name, continent, wikipedia_link, keywords FROM countries ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	countries := []Country{}
	for rows.Next() {
		var id sql.NullInt64
		var code, name, continent, wikipediaLink, keywords sql.NullString
		if err := rows.Scan(&id, &code, &name, &continent, &wikipediaLink, &keywords); err != nil {
			return nil, err
		}
		countries = append(countries, Country{
			ID:            int(id.Int64),
			Code:          code.String,
			Name:          name.String,
			Continent:     continent.String,
			WikipediaLink: wikipediaLink.String,
			Keywords:      keywords.String,
		})
	}
	return countries, rows.Err()
}

// ImportStatus describes the data behind each imported table.
func (s *Store) ImportStatus() ([]ImportStatus, error) {
	rows, err := s.db.Query(`SELECT table_name, last_import_date, git_commit_hash, git_commit_date, record_count
		FROM import_status
		ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []ImportStatus{}
	for rows.Next() {
		var st ImportStatus
		if err := rows.Scan(&st.TableName, &st.LastImportDate, &st.GitCommitHash, &st.GitCommitDate, &st.RecordCount); err != nil {
			return nil, err
		}
		tables = append(tables, st)
	}
	return tables, rows.Err()
}

// airportsInRange finds all airports within rangeNM nautical miles of the
// origin airport, nearest first.
func (s *Store) airportsInRange(origin *Airport, rangeNM float64, types []string) ([]ReachableAirport, error) {
	box := boundingBoxAround(origin.LatitudeDeg, origin.LongitudeDeg, rangeNM)

	query := `SELECT ` + airportColumns + ` FROM airports
		WHERE id != ? AND latitude_deg BETWEEN ? AND ?`
	args := []any{origin.ID, box.MinLat, box.MaxLat}

	switch {
	case box.AllLon:
	case box.MinLon < -180:
		// Antimeridian crossing: the box wraps to the eastern edge.
		query += " AND (longitude_deg >= ? OR longitude_deg <= ?)"
		args = append(args, box.MinLon+360, box.MaxLon)
	case box.MaxLon > 180:
		query += " AND (longitude_deg >= ? OR longitude_deg <= ?)"
		args = append(args, box.MinLon, box.MaxLon-360)
	default:
		query += " AND longitude_deg BETWEEN ? AND ?"
		args = append(args, box.MinLon, box.MaxLon)
	}

	if len(types) > 0 {
		query += " AND type IN (" + strings.TrimSuffix(strings.Repeat("?,", len(types)), ",") + ")"
		for _, t := range types {
			args = append(args, t)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	candidates, err := scanAirports(rows)
	if err != nil {
		return nil, err
	}

	results := []ReachableAirport{}
	for _, a := range candidates {
		dist := calculateDistance(origin.LatitudeDeg, origin.LongitudeDeg, a.LatitudeDeg, a.LongitudeDeg)
		if dist <= rangeNM {
			results = append(results, ReachableAirport{Airport: a, DistanceNM: roundTo(dist, 1)})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceNM < results[j].DistanceNM
	})
	return results, nil
}
