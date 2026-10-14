package server

import (
	"database/sql"
	"sort"
	"strings"
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

// getAirportByCode looks an airport up by its 4-letter code. Most airports in
// the dataset have no icao_code, only an ident or GPS code that is the ICAO
// code in practice (FVHA, Harare), so those are tried next. Ties go to the
// most significant airport, which skips "closed" duplicates.
func (s *Server) getAirportByCode(code string) (*Airport, error) {
	code = strings.ToUpper(code)
	row := s.db.QueryRow(`SELECT `+airportColumns+` FROM airports
		WHERE icao_code = ?1 OR ident = ?1 OR gps_code = ?1
		ORDER BY CASE WHEN icao_code = ?1 THEN 0 WHEN ident = ?1 THEN 1 ELSE 2 END, `+typeRank+`
		LIMIT 1`, code)

	a, err := scanAirport(row)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// searchAirports finds airports whose name, city or keywords contain query,
// or whose code is exactly query. Exact code matches come first. At most
// limit airports are returned; truncated reports whether more matched.
func (s *Server) searchAirports(query, country string, limit int) (airports []Airport, truncated bool, err error) {
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
		return nil, false, err
	}
	airports, err = scanAirports(rows)
	if err != nil {
		return nil, false, err
	}

	if len(airports) > limit {
		return airports[:limit], true, nil
	}
	return airports, false, nil
}

// getAirportsInRange finds all airports within rangeNM nautical miles of the origin airport.
// If types is non-empty, only airports matching those types are returned.
func (s *Server) getAirportsInRange(origin *Airport, rangeNM float64, types []string) ([]ReachableAirport, error) {
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
