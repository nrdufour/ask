package store

import (
	"errors"
	"testing"
)

func TestDetail(t *testing.T) {
	s := newTestStore(t)

	d, err := s.Detail("lfpg")
	if err != nil {
		t.Fatal(err)
	}
	if d.CountryName != "France" || d.RegionName != "Île-de-France" {
		t.Errorf("country %q, region %q", d.CountryName, d.RegionName)
	}

	if len(d.Runways) != 5 {
		t.Fatalf("got %d LFPG runways, want 5", len(d.Runways))
	}
	// Longest open runway first: 09R/27L, 13,829 ft with headings in the data.
	r := d.Runways[0]
	if r.LowEnd.Ident == "" || *r.LengthFt < 13000 || r.LowEnd.HeadingSource != HeadingFromData {
		t.Errorf("first runway %s/%s, %d ft, heading from %q", r.LowEnd.Ident, r.HighEnd.Ident, *r.LengthFt, r.LowEnd.HeadingSource)
	}
	if r.SurfaceLabel == "" || !r.Lighted {
		t.Errorf("first runway surface %q lighted %v", r.SurfaceLabel, r.Lighted)
	}

	if len(d.Frequencies) == 0 || d.Frequencies[0].Type != "ATIS" {
		t.Errorf("frequencies should start with ATIS: %+v", d.Frequencies)
	}

	// Of the worldwide navaids named BT, only Le Bourget is near LFPG.
	var bt []string
	for i, n := range d.Navaids {
		if n.DistanceNM > NavaidRadiusNM {
			t.Errorf("%s is %.1f NM away, beyond %d", n.Ident, n.DistanceNM, NavaidRadiusNM)
		}
		if i > 0 && n.DistanceNM < d.Navaids[i-1].DistanceNM {
			t.Errorf("navaids not sorted by distance at %s", n.Ident)
		}
		if n.Ident == "BT" {
			bt = append(bt, n.Name)
		}
	}
	if len(bt) != 1 || bt[0] != "Le Bourget" {
		t.Errorf("BT navaids near LFPG: %v, want only Le Bourget", bt)
	}
}

func TestDetailByIdent(t *testing.T) {
	s := newTestStore(t)

	d, err := s.Detail("xx-0001")
	if err != nil {
		t.Fatal(err)
	}
	if d.Airport.Ident != "XX-0001" {
		t.Errorf("got %s", d.Airport.Ident)
	}

	if _, err := s.Detail("XX-9999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown ident: err = %v, want ErrNotFound", err)
	}
	var inputErr *InputError
	if _, err := s.Detail("not a code!"); !errors.As(err, &inputErr) {
		t.Errorf("bad code: err = %v, want an InputError", err)
	}
}

func TestRunwayHeadingFallbacks(t *testing.T) {
	s := newTestStore(t)

	d, err := s.Detail("XX-0001")
	if err != nil {
		t.Fatal(err)
	}
	byIdent := map[string]RunwayEnd{}
	for _, r := range d.Runways {
		byIdent[r.LowEnd.Ident] = r.LowEnd
		byIdent[r.HighEnd.Ident] = r.HighEnd
	}

	tests := []struct {
		ident  string
		source string
		want   float64
	}{
		{"09", HeadingFromThresholds, 90}, // due east between the two thresholds
		{"27", HeadingFromThresholds, 270},
		{"18L", HeadingFromDesignator, 180},
		{"36R", HeadingFromDesignator, 360},
		{"H1", "", 0},
	}
	for _, tt := range tests {
		end, ok := byIdent[tt.ident]
		if !ok {
			t.Errorf("no runway end %s", tt.ident)
			continue
		}
		if end.HeadingSource != tt.source {
			t.Errorf("%s: heading from %q, want %q", tt.ident, end.HeadingSource, tt.source)
		}
		if tt.source == "" {
			if end.HeadingDeg != nil {
				t.Errorf("%s: heading %v, want none", tt.ident, *end.HeadingDeg)
			}
			continue
		}
		if end.HeadingDeg == nil || *end.HeadingDeg < tt.want-0.5 || *end.HeadingDeg > tt.want+0.5 {
			t.Errorf("%s: heading %v, want %v", tt.ident, end.HeadingDeg, tt.want)
		}
	}
}

func TestSurfaceLabel(t *testing.T) {
	for code, want := range map[string]string{
		"ASP": "asphalt", "ASPH-G": "asphalt", "CONC": "concrete", "CON": "concrete",
		"TURF-F": "grass", "GRS": "grass", "Grass / Sod": "grass", "GVL": "gravel",
		"WATER": "water", "X": "x",
	} {
		if got := surfaceLabel(code); got != want {
			t.Errorf("surfaceLabel(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestLocalTimeByIdent(t *testing.T) {
	s := newTestStore(t)
	got, err := s.LocalTime("XX-0001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Timezone != "Europe/Paris" {
		t.Errorf("timezone %s, want Europe/Paris", got.Timezone)
	}
}
