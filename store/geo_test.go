package store

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestCalculateDistance(t *testing.T) {
	// KJFK to EGLL is 2999 NM on the WGS84 ellipsoid. The spherical model
	// should stay within 0.5% of that.
	d := calculateDistance(40.6398, -73.7789, 51.4706, -0.461941)
	if math.Abs(d-2999)/2999 > 0.005 {
		t.Errorf("KJFK-EGLL = %.1f NM, want within 0.5%% of 2999", d)
	}

	// A quarter of a great circle.
	d = calculateDistance(0, 0, 0, 90)
	if want := earthRadiusNM * math.Pi / 2; math.Abs(d-want) > 1e-6 {
		t.Errorf("quarter circle = %f NM, want %f", d, want)
	}
}

func TestInitialBearing(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		want                   float64
	}{
		{"north", 0, 0, 10, 0, 0},
		{"east", 0, 0, 0, 10, 90},
		{"south", 10, 0, 0, 0, 180},
		{"west", 0, 10, 0, 0, 270},
		{"across antimeridian is east", 0, 179, 0, -179, 90},
		// The great circle from Paris to New York leaves heading north-west.
		{"LFPG-KJFK", 49.0097, 2.5479, 40.6398, -73.7789, 291.6},
	}
	for _, tt := range tests {
		got := initialBearing(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
		if math.Abs(got-tt.want) > 0.1 {
			t.Errorf("%s: bearing = %.2f, want %.1f", tt.name, got, tt.want)
		}
	}
}

// destination returns the point reached from (lat, lon) after distNM on the
// given true course.
func destination(lat, lon, bearing, distNM float64) (float64, float64) {
	phi, lambda, theta := toRad(lat), toRad(lon), toRad(bearing)
	d := distNM / earthRadiusNM

	phi2 := math.Asin(math.Sin(phi)*math.Cos(d) + math.Cos(phi)*math.Sin(d)*math.Cos(theta))
	lambda2 := lambda + math.Atan2(math.Sin(theta)*math.Sin(d)*math.Cos(phi), math.Cos(d)-math.Sin(phi)*math.Sin(phi2))

	lon2 := math.Mod(toDeg(lambda2)+540, 360) - 180
	return toDeg(phi2), lon2
}

func (b boundingBox) contains(lat, lon float64) bool {
	const eps = 1e-9
	if lat < b.MinLat-eps || lat > b.MaxLat+eps {
		return false
	}
	if b.AllLon {
		return true
	}
	for _, l := range []float64{lon - 360, lon, lon + 360} {
		if l >= b.MinLon-eps && l <= b.MaxLon+eps {
			return true
		}
	}
	return false
}

// Every point within range must fall in the box. The previous d / cos φ
// half-width failed this at high latitudes and long ranges.
func TestBoundingBoxContainsEveryPointInRange(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 200_000; i++ {
		lat := rng.Float64()*180 - 90
		lon := rng.Float64()*360 - 180
		rangeNM := rng.Float64() * 3000
		box := boundingBoxAround(lat, lon, rangeNM)

		pLat, pLon := destination(lat, lon, rng.Float64()*360, rng.Float64()*rangeNM)
		if !box.contains(pLat, pLon) {
			t.Fatalf("origin (%.4f, %.4f) range %.1f NM: point (%.4f, %.4f) outside %+v",
				lat, lon, rangeNM, pLat, pLon, box)
		}
	}
}

func TestBoundingBoxAroundPole(t *testing.T) {
	// ENSB (Svalbard) is 707 NM from the North Pole.
	if box := boundingBoxAround(78.2461, 15.4656, 1000); !box.AllLon || box.MaxLat != 90 {
		t.Errorf("1000 NM from ENSB should cover every longitude up to the pole, got %+v", box)
	}
	if box := boundingBoxAround(78.2461, 15.4656, 500); box.AllLon {
		t.Errorf("500 NM from ENSB should not reach the pole, got %+v", box)
	}
}
