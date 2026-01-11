package postgresosm

import (
	"encoding/json"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/location-microservice/internal/domain"
)

func hashCategory(code string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(code))
	return int64(h.Sum64())
}

func ensureName(name string, category string, osmID int64) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return strings.Title(category) + " " + strconv.FormatInt(osmID, 10)
}

func parseTags(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}

	var tmp map[string]string
	if err := json.Unmarshal(raw, &tmp); err != nil {
		return map[string]string{}
	}

	return tmp
}

func pickTag(tags map[string]string, keys ...string) *string {
	for _, key := range keys {
		if val, ok := tags[key]; ok && strings.TrimSpace(val) != "" {
			value := strings.TrimSpace(val)
			return &value
		}
	}
	return nil
}

func parseBoolTag(tags map[string]string, keys ...string) *bool {
	for _, key := range keys {
		if val, ok := tags[key]; ok {
			if b, okParsed := parseYesNo(val); okParsed {
				return &b
			}
		}
	}
	return nil
}

func parseYesNo(val string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "yes", "true", "1", "y":
		return true, true
	case "no", "false", "0", "n":
		return false, true
	default:
		return false, false
	}
}

func parseIntTag(tags map[string]string, keys ...string) *int {
	for _, key := range keys {
		if val, ok := tags[key]; ok && strings.TrimSpace(val) != "" {
			if parsed, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

func parsePOIFromRow(row *poiRow) *domain.POI {
	tags := parseTags(row.TagsJSON)

	poi := &domain.POI{
		ID:          row.OSMID,
		OSMId:       row.OSMID,
		Name:        ensureName(row.Name, row.Category, row.OSMID),
		Category:    row.Category,
		Subcategory: row.Subcategory,
		Lat:         row.Lat,
		Lon:         row.Lon,
		Geometry:    row.Geometry,
		Tags:        tags,
	}

	poi.NameEn = pickTag(tags, "name:en")
	poi.NameEs = pickTag(tags, "name:es")
	poi.NameCa = pickTag(tags, "name:ca")
	poi.NameRu = pickTag(tags, "name:ru")
	poi.NameUk = pickTag(tags, "name:uk")
	poi.NameFr = pickTag(tags, "name:fr")
	poi.NamePt = pickTag(tags, "name:pt")
	poi.NameIt = pickTag(tags, "name:it")
	poi.NameDe = pickTag(tags, "name:de")

	poi.Address = pickTag(tags, "addr:full", "addr:street", "addr:place")
	poi.Phone = pickTag(tags, "phone", "contact:phone")
	poi.Website = pickTag(tags, "website", "contact:website", "url")
	poi.Email = pickTag(tags, "email", "contact:email")
	poi.OpeningHours = pickTag(tags, "opening_hours")
	poi.Description = pickTag(tags, "description", "note")
	poi.Brand = pickTag(tags, "brand", "brand:wikidata")
	poi.Operator = pickTag(tags, "operator", "operator:wikidata")
	poi.Cuisine = pickTag(tags, "cuisine")
	poi.Diet = pickTag(tags, "diet", "diet:vegetarian", "diet:vegan")
	poi.Smoking = pickTag(tags, "smoking")
	poi.Internet = pickTag(tags, "internet_access", "internet", "wifi")
	poi.Fee = parseBoolTag(tags, "fee")
	poi.Charge = pickTag(tags, "charge", "fee:conditional")
	poi.PaymentCash = parseBoolTag(tags, "payment:cash")
	poi.PaymentCards = parseBoolTag(tags, "payment:cards", "payment:credit_cards", "payment:debit_cards")
	poi.Outdoor = parseBoolTag(tags, "outdoor_seating")
	poi.Takeaway = parseBoolTag(tags, "takeaway")
	poi.Delivery = parseBoolTag(tags, "delivery")
	poi.DriveThrough = parseBoolTag(tags, "drive_through")
	poi.Wheelchair = parseBoolTag(tags, "wheelchair")
	poi.InternetFee = parseBoolTag(tags, "internet_access:fee", "internet:fee")
	poi.Stars = parseIntTag(tags, "stars", "tourism:stars")
	poi.Rooms = parseIntTag(tags, "rooms")
	poi.Beds = parseIntTag(tags, "beds", "capacity:beds")
	poi.Capacity = parseIntTag(tags, "capacity", "capacity:persons")
	poi.MinAge = parseIntTag(tags, "min_age")
	poi.Facebook = pickTag(tags, "facebook", "contact:facebook")
	poi.Instagram = pickTag(tags, "instagram", "contact:instagram")
	poi.Twitter = pickTag(tags, "twitter", "contact:twitter")
	poi.ImageUrl = pickTag(tags, "image", "wikimedia_commons")
	poi.Wikidata = pickTag(tags, "wikidata")
	poi.Wikipedia = pickTag(tags, "wikipedia")

	// OSM специфичные поля
	poi.Religion = pickTag(tags, "religion")
	poi.Denomination = pickTag(tags, "denomination")
	poi.Sport = pickTag(tags, "sport")
	poi.Building = pickTag(tags, "building")
	poi.Historic = pickTag(tags, "historic")
	poi.Military = pickTag(tags, "military")
	poi.Office = pickTag(tags, "office")

	return poi
}

func getPOILimitByZoom(zoom int) int {
	switch {
	case zoom < 10:
		return 50
	case zoom < 13:
		return 200
	case zoom < 15:
		return 500
	default:
		return 1000
	}
}

// scanInt64Array парсит PostgreSQL array bigint[] в []int64
func scanInt64Array(src interface{}) ([]int64, error) {
	if src == nil {
		return []int64{}, nil
	}

	switch v := src.(type) {
	case []byte:
		str := string(v)
		if str == "{}" || str == "" {
			return []int64{}, nil
		}

		str = strings.Trim(str, "{}")
		if str == "" {
			return []int64{}, nil
		}

		parts := strings.Split(str, ",")
		result := make([]int64, 0, len(parts))

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var val int64
			if _, err := strconv.ParseInt(part, 10, 64); err == nil {
				val, _ = strconv.ParseInt(part, 10, 64)
				result = append(result, val)
			}
		}
		return result, nil
	default:
		return []int64{}, nil
	}
}

// int64ArrayToString конвертирует []int64 в формат PostgreSQL array
func int64ArrayToString(arr []int64) string {
	if len(arr) == 0 {
		return "{}"
	}

	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return "{" + strings.Join(strs, ",") + "}"
}
