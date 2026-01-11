package postgresosm

const (
	SRID4326  = 4326
	SRID3857  = 3857
	MVTExtent = 4096
	MVTBuffer = 256

	LimitPOIs             = 100
	LimitPOIsRadius       = 200
	LimitPOIsCategory     = 1000
	LimitStations         = 100
	LimitLines            = 50
	LimitGreenSpaces      = 50
	LimitWaterBodies      = 50
	LimitBeaches          = 20
	LimitNoiseSources     = 50
	LimitTouristZones     = 50
	LimitBoundaries       = 100
	LimitBoundariesRadius = 50

	// BoundaryExpansionDegrees - расширение для поиска границ (~11км на экваторе)
	BoundaryExpansionDegrees = 0.1
)

const (
	planetPointTable   = "planet_osm_point"
	planetLineTable    = "planet_osm_line"
	planetPolygonTable = "planet_osm_polygon"
	planetRoadsTable   = "planet_osm_roads"
)

// expressions для повторного использования в SQL
const (
	// categoryExpr - определяет категорию POI на основе OSM тегов (приоритет слева направо)
	categoryExpr = `COALESCE(NULLIF(amenity,''), NULLIF(shop,''), NULLIF(tourism,''), NULLIF(leisure,''), NULLIF(historic,''), NULLIF(office,''), NULLIF(man_made,''), NULLIF("natural",''), NULLIF(highway,''), NULLIF(public_transport,''), NULLIF(railway,''), NULLIF(aeroway,''), NULLIF(military,''), NULLIF(place,''), 'other')`

	// subcategoryExpr - определяет подкатегорию из дополнительных тегов
	subcategoryExpr = "COALESCE(NULLIF(tags->'cuisine',''), NULLIF(tags->'sport',''), NULLIF(tags->'religion',''), NULLIF(tags->'denomination',''), NULLIF(tags->'building',''), NULLIF(shop,''), NULLIF(tourism,''), 'general')"
)
