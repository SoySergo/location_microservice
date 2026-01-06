package errors

import "net/http"

var (
	ErrLocationNotFound = New(
		"LOCATION_NOT_FOUND",
		"Location not found",
		http.StatusNotFound,
	)

	ErrInvalidCoordinates = New(
		"INVALID_COORDINATES",
		"Invalid coordinates provided",
		http.StatusBadRequest,
	)

	ErrInvalidRadius = New(
		"INVALID_RADIUS",
		"Invalid radius value",
		http.StatusBadRequest,
	)

	ErrInvalidZoom = New(
		"INVALID_ZOOM",
		"Invalid zoom level",
		http.StatusBadRequest,
	)

	ErrInvalidTileCoordinates = New(
		"INVALID_TILE_COORDINATES",
		"Invalid tile coordinates",
		http.StatusBadRequest,
	)

	ErrInvalidBoundaryID = New(
		"INVALID_BOUNDARY_ID",
		"Invalid boundary ID",
		http.StatusBadRequest,
	)

	ErrDatabaseError = New(
		"DATABASE_ERROR",
		"Database operation failed",
		http.StatusInternalServerError,
	)

	ErrCacheError = New(
		"CACHE_ERROR",
		"Cache operation failed",
		http.StatusInternalServerError,
	)

	ErrInvalidRequest = New(
		"INVALID_REQUEST",
		"Invalid request parameters",
		http.StatusBadRequest,
	)

	ErrInternalServer = New(
		"INTERNAL_SERVER_ERROR",
		"Internal server error",
		http.StatusInternalServerError,
	)
)

var (
ErrInvalidTransportType = New(
"INVALID_TRANSPORT_TYPE",
"Invalid transport type",
http.StatusBadRequest,
)

CodeInvalidInput = "INVALID_INPUT"
)
