package server

import "math"

const (
	// Earth's mean radius in nautical miles
	earthRadiusNM = 3440.065

	nmToKM = 1.852
	nmToSM = 1.852 / 1.609344
)

func toRad(deg float64) float64 { return deg * math.Pi / 180 }
func toDeg(rad float64) float64 { return rad * 180 / math.Pi }

// calculateDistance computes the great circle distance between two points using the haversine formula
// Returns distance in nautical miles
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad, lat2Rad := toRad(lat1), toRad(lat2)
	dLat := lat2Rad - lat1Rad
	dLon := toRad(lon2 - lon1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusNM * c
}

// initialBearing returns the true course in degrees [0, 360) to fly from the
// first point to the second along the great circle.
func initialBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad, lat2Rad := toRad(lat1), toRad(lat2)
	dLon := toRad(lon2 - lon1)

	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)

	return math.Mod(toDeg(math.Atan2(y, x))+360, 360)
}

// boundingBox is a lat/lon rectangle enclosing a spherical cap. When the cap
// crosses the antimeridian, MinLon < -180 or MaxLon > 180.
type boundingBox struct {
	MinLat, MaxLat float64
	MinLon, MaxLon float64
	// AllLon is set when the cap contains a pole: every longitude is in range.
	AllLon bool
}

// boundingBoxAround returns the smallest lat/lon box containing every point
// within rangeNM of (lat, lon).
//
// The longitude half-width is asin(sin d / cos φ), d being the angular
// radius: the meridians tangent to the cap. The naive d / cos φ is narrower,
// and misses airports at high latitude or long range.
func boundingBoxAround(lat, lon, rangeNM float64) boundingBox {
	d := rangeNM / earthRadiusNM
	phi := toRad(lat)

	minLat, maxLat := phi-d, phi+d
	if minLat <= -math.Pi/2 || maxLat >= math.Pi/2 {
		return boundingBox{
			MinLat: toDeg(math.Max(minLat, -math.Pi/2)),
			MaxLat: toDeg(math.Min(maxLat, math.Pi/2)),
			MinLon: -180,
			MaxLon: 180,
			AllLon: true,
		}
	}

	// |φ| + d < π/2 here, so sin d < cos φ and the asin argument is below 1.
	dLon := toDeg(math.Asin(math.Sin(d) / math.Cos(phi)))
	return boundingBox{
		MinLat: toDeg(minLat),
		MaxLat: toDeg(maxLat),
		MinLon: lon - dLon,
		MaxLon: lon + dLon,
	}
}

func roundTo(x float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(x*p) / p
}
