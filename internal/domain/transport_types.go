package domain

import "strings"

// Transport type constants
const (
	TransportTypeMetro        = "metro"
	TransportTypeBus          = "bus"
	TransportTypeTram         = "tram"
	TransportTypeCercania     = "cercania"
	TransportTypeLongDistance = "long_distance"
	TransportTypeTrain        = "train" // Generic train type (existing in DB)
)

// Transport type mapping for OSM data
var OSMTagToTransportType = map[string]map[string]string{
	"railway": {
		"subway":      TransportTypeMetro,
		"tram_stop":   TransportTypeTram,
		"station":     TransportTypeTrain, // Will be further classified by network
	},
	"highway": {
		"bus_stop": TransportTypeBus,
	},
	"route": {
		"bus":  TransportTypeBus,
		"tram": TransportTypeTram,
	},
	"station": {
		"subway": TransportTypeMetro,
	},
}

// Network patterns for classifying train stations
var NetworkPatterns = map[string]string{
	"Rodalies":  TransportTypeCercania,
	"Cercanías": TransportTypeCercania,
	"Renfe":     TransportTypeLongDistance,
	"AVE":       TransportTypeLongDistance,
}

// ValidTransportTypes returns list of valid transport types
func ValidTransportTypes() []string {
	return []string{
		TransportTypeMetro,
		TransportTypeBus,
		TransportTypeTram,
		TransportTypeCercania,
		TransportTypeLongDistance,
	}
}

// IsValidTransportType checks if transport type is valid
func IsValidTransportType(transportType string) bool {
	validTypes := ValidTransportTypes()
	for _, t := range validTypes {
		if t == transportType {
			return true
		}
	}
	return false
}

// ClassifyTrainStation classifies a train station based on network
func ClassifyTrainStation(network string) string {
	if network == "" {
		return TransportTypeTrain
	}
	
	for pattern, transportType := range NetworkPatterns {
		if strings.Contains(network, pattern) {
			return transportType
		}
	}
	
	return TransportTypeTrain
}
