package domain

import "time"

// POI представляет точку интереса
type POI struct {
	ID           int64   `json:"id" db:"id"`
	OSMId        int64   `json:"osm_id" db:"osm_id"`
	Name         string  `json:"name" db:"name"`
	NameEn       *string `json:"name_en,omitempty" db:"name_en"`
	NameEs       *string `json:"name_es,omitempty" db:"name_es"`
	NameCa       *string `json:"name_ca,omitempty" db:"name_ca"`
	NameRu       *string `json:"name_ru,omitempty" db:"name_ru"`
	NameUk       *string `json:"name_uk,omitempty" db:"name_uk"`
	NameFr       *string `json:"name_fr,omitempty" db:"name_fr"`
	NamePt       *string `json:"name_pt,omitempty" db:"name_pt"`
	NameIt       *string `json:"name_it,omitempty" db:"name_it"`
	NameDe       *string `json:"name_de,omitempty" db:"name_de"`
	Category     string  `json:"category" db:"category"`
	Subcategory  string  `json:"subcategory" db:"subcategory"`
	Lat          float64 `json:"lat" db:"lat"`
	Lon          float64 `json:"lon" db:"lon"`
	Geometry     []byte  `json:"-" db:"geometry"`
	Address      *string `json:"address,omitempty" db:"address"`
	Phone        *string `json:"phone,omitempty" db:"phone"`
	Website      *string `json:"website,omitempty" db:"website"`
	Email        *string `json:"email,omitempty" db:"email"`
	OpeningHours *string `json:"opening_hours,omitempty" db:"opening_hours"`
	Wheelchair   *bool   `json:"wheelchair,omitempty" db:"wheelchair"`

	// Дополнительная информация
	Description *string `json:"description,omitempty" db:"description"`
	Brand       *string `json:"brand,omitempty" db:"brand"`
	Operator    *string `json:"operator,omitempty" db:"operator"`
	Cuisine     *string `json:"cuisine,omitempty" db:"cuisine"`
	Diet        *string `json:"diet,omitempty" db:"diet"`
	Stars       *int    `json:"stars,omitempty" db:"stars"`
	Rooms       *int    `json:"rooms,omitempty" db:"rooms"`
	Beds        *int    `json:"beds,omitempty" db:"beds"`
	Capacity    *int    `json:"capacity,omitempty" db:"capacity"`
	MinAge      *int    `json:"min_age,omitempty" db:"min_age"`

	// OSM специфичные поля
	Religion     *string `json:"religion,omitempty" db:"religion"`
	Denomination *string `json:"denomination,omitempty" db:"denomination"`
	Sport        *string `json:"sport,omitempty" db:"sport"`
	Building     *string `json:"building,omitempty" db:"building"`
	Historic     *string `json:"historic,omitempty" db:"historic"`
	Military     *string `json:"military,omitempty" db:"military"`
	Office       *string `json:"office,omitempty" db:"office"`

	// Услуги и удобства
	Internet     *string `json:"internet,omitempty" db:"internet"`
	InternetFee  *bool   `json:"internet_fee,omitempty" db:"internet_fee"`
	Smoking      *string `json:"smoking,omitempty" db:"smoking"`
	Outdoor      *bool   `json:"outdoor_seating,omitempty" db:"outdoor_seating"`
	Takeaway     *bool   `json:"takeaway,omitempty" db:"takeaway"`
	Delivery     *bool   `json:"delivery,omitempty" db:"delivery"`
	DriveThrough *bool   `json:"drive_through,omitempty" db:"drive_through"`

	// Оплата
	Fee          *bool   `json:"fee,omitempty" db:"fee"`
	Charge       *string `json:"charge,omitempty" db:"charge"`
	PaymentCash  *bool   `json:"payment_cash,omitempty" db:"payment_cash"`
	PaymentCards *bool   `json:"payment_cards,omitempty" db:"payment_cards"`

	// Социальные сети и контакты
	Facebook  *string `json:"facebook,omitempty" db:"facebook"`
	Instagram *string `json:"instagram,omitempty" db:"instagram"`
	Twitter   *string `json:"twitter,omitempty" db:"twitter"`

	// Дополнительные поля
	ImageUrl  *string `json:"image_url,omitempty" db:"image_url"`
	Wikidata  *string `json:"wikidata,omitempty" db:"wikidata"`
	Wikipedia *string `json:"wikipedia,omitempty" db:"wikipedia"`

	Tags         map[string]string `json:"tags,omitempty" db:"tags"`
	SearchVector string            `json:"-" db:"search_vector"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// POICategory представляет категорию POI
type POICategory struct {
	ID        int64     `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	NameEn    string    `json:"name_en" db:"name_en"`
	NameEs    string    `json:"name_es" db:"name_es"`
	NameCa    string    `json:"name_ca" db:"name_ca"`
	NameRu    string    `json:"name_ru" db:"name_ru"`
	NameUk    string    `json:"name_uk" db:"name_uk"`
	NameFr    string    `json:"name_fr" db:"name_fr"`
	NamePt    string    `json:"name_pt" db:"name_pt"`
	NameIt    string    `json:"name_it" db:"name_it"`
	NameDe    string    `json:"name_de" db:"name_de"`
	Icon      *string   `json:"icon,omitempty" db:"icon"`
	Color     *string   `json:"color,omitempty" db:"color"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// POISubcategory представляет подкатегорию POI
type POISubcategory struct {
	ID         int64     `json:"id" db:"id"`
	CategoryID int64     `json:"category_id" db:"category_id"`
	Code       string    `json:"code" db:"code"`
	NameEn     string    `json:"name_en" db:"name_en"`
	NameEs     string    `json:"name_es" db:"name_es"`
	NameCa     string    `json:"name_ca" db:"name_ca"`
	NameRu     string    `json:"name_ru" db:"name_ru"`
	NameUk     string    `json:"name_uk" db:"name_uk"`
	NameFr     string    `json:"name_fr" db:"name_fr"`
	NamePt     string    `json:"name_pt" db:"name_pt"`
	NameIt     string    `json:"name_it" db:"name_it"`
	NameDe     string    `json:"name_de" db:"name_de"`
	Icon       *string   `json:"icon,omitempty" db:"icon"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
