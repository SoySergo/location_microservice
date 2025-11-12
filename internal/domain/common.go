package domain

type Point struct {
	Lat float64 `json:"lat" db:"lat"`
	Lon float64 `json:"lon" db:"lon"`
}

type BoundingBox struct {
	MinLat float64 `json:"min_lat" db:"min_lat"`
	MinLon float64 `json:"min_lon" db:"min_lon"`
	MaxLat float64 `json:"max_lat" db:"max_lat"`
	MaxLon float64 `json:"max_lon" db:"max_lon"`
}
