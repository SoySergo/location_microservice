package domain

// NearbyCategoryMapping отображает фронтенд-категории фильтров на значения OSM-тегов.
// Ключ — категория фронтенда (transport, schools, medical, ...),
// значение — список OSM tag values для POI поиска через categoryExpr.
var NearbyCategoryMapping = map[string][]string{
	"schools":       {"school", "kindergarten", "college", "university", "library", "language_school"},
	"medical":       {"pharmacy", "hospital", "clinic", "doctors", "dentist", "veterinary"},
	"groceries":     {"supermarket", "convenience", "grocery", "bakery", "butcher", "greengrocer"},
	"shopping":      {"mall", "department_store", "clothes", "shoes", "electronics", "furniture", "jewelry"},
	"restaurants":   {"restaurant", "cafe", "bar", "fast_food"},
	"sports":        {"sports_centre", "fitness_centre", "swimming_pool", "stadium"},
	"entertainment": {"cinema", "theatre", "nightclub", "casino"},
	"parks":         {"park", "garden", "playground"},
	"beauty":        {"hairdresser", "beauty"},
	"attractions":   {"attraction", "viewpoint", "museum", "gallery", "monument", "castle", "archaeological_site"},
}

// TransportCategory — специальный ключ категории для транспорта (обрабатывается отдельно)
const TransportCategory = "transport"

// IsValidNearbyCategory проверяет, является ли категория допустимой
func IsValidNearbyCategory(category string) bool {
	if category == TransportCategory {
		return true
	}
	_, ok := NearbyCategoryMapping[category]
	return ok
}

// GetOSMCategories возвращает список OSM-тегов для фронтенд-категории
func GetOSMCategories(category string) []string {
	tags, ok := NearbyCategoryMapping[category]
	if !ok {
		return nil
	}
	return tags
}
