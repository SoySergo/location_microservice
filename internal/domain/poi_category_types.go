package domain

// POI Category constants
const (
	POICategoryHealthcare = "healthcare"
	POICategoryShopping   = "shopping"
	POICategoryEducation  = "education"
	POICategoryLeisure    = "leisure"
	POICategoryFoodDrink  = "food_drink"
)

// POI Subcategory constants for Healthcare
const (
	POISubcategoryPharmacy    = "pharmacy"
	POISubcategoryHospital    = "hospital"
	POISubcategoryClinic      = "clinic"
	POISubcategoryDoctors     = "doctors"
	POISubcategoryDentist     = "dentist"
	POISubcategoryVeterinary  = "veterinary"
)

// POI Subcategory constants for Shopping
const (
	POISubcategorySupermarket     = "supermarket"
	POISubcategoryConvenience     = "convenience"
	POISubcategoryMall            = "mall"
	POISubcategoryGrocery         = "grocery"
	POISubcategoryDepartmentStore = "department_store"
	POISubcategoryBakery          = "bakery"
	POISubcategoryButcher         = "butcher"
	POISubcategoryGreengrocer     = "greengrocer"
)

// POI Subcategory constants for Education
const (
	POISubcategorySchool         = "school"
	POISubcategoryKindergarten   = "kindergarten"
	POISubcategoryCollege        = "college"
	POISubcategoryUniversity     = "university"
	POISubcategoryLibrary        = "library"
	POISubcategoryLanguageSchool = "language_school"
)

// POI Subcategory constants for Leisure
const (
	POISubcategoryPark          = "park"
	POISubcategoryGarden        = "garden"
	POISubcategoryPlayground    = "playground"
	POISubcategorySportsCentre  = "sports_centre"
	POISubcategoryAttraction    = "attraction"
	POISubcategoryViewpoint     = "viewpoint"
	POISubcategoryMuseum        = "museum"
	POISubcategoryMonument      = "monument"
	POISubcategoryCastle        = "castle"
)

// POI Subcategory constants for Food & Drink
const (
	POISubcategoryRestaurant = "restaurant"
	POISubcategoryCafe       = "cafe"
	POISubcategoryBar        = "bar"
	POISubcategoryFastFood   = "fast_food"
)

// OSMTagToPOICategory maps OSM tags to POI categories and subcategories
var OSMTagToPOICategory = map[string]map[string]CategoryMapping{
	"amenity": {
		"pharmacy":       {Category: POICategoryHealthcare, Subcategory: POISubcategoryPharmacy},
		"hospital":       {Category: POICategoryHealthcare, Subcategory: POISubcategoryHospital},
		"clinic":         {Category: POICategoryHealthcare, Subcategory: POISubcategoryClinic},
		"doctors":        {Category: POICategoryHealthcare, Subcategory: POISubcategoryDoctors},
		"dentist":        {Category: POICategoryHealthcare, Subcategory: POISubcategoryDentist},
		"veterinary":     {Category: POICategoryHealthcare, Subcategory: POISubcategoryVeterinary},
		"school":         {Category: POICategoryEducation, Subcategory: POISubcategorySchool},
		"kindergarten":   {Category: POICategoryEducation, Subcategory: POISubcategoryKindergarten},
		"college":        {Category: POICategoryEducation, Subcategory: POISubcategoryCollege},
		"university":     {Category: POICategoryEducation, Subcategory: POISubcategoryUniversity},
		"library":        {Category: POICategoryEducation, Subcategory: POISubcategoryLibrary},
		"language_school":{Category: POICategoryEducation, Subcategory: POISubcategoryLanguageSchool},
		"restaurant":     {Category: POICategoryFoodDrink, Subcategory: POISubcategoryRestaurant},
		"cafe":           {Category: POICategoryFoodDrink, Subcategory: POISubcategoryCafe},
		"bar":            {Category: POICategoryFoodDrink, Subcategory: POISubcategoryBar},
		"fast_food":      {Category: POICategoryFoodDrink, Subcategory: POISubcategoryFastFood},
	},
	"shop": {
		"supermarket":      {Category: POICategoryShopping, Subcategory: POISubcategorySupermarket},
		"convenience":      {Category: POICategoryShopping, Subcategory: POISubcategoryConvenience},
		"mall":             {Category: POICategoryShopping, Subcategory: POISubcategoryMall},
		"grocery":          {Category: POICategoryShopping, Subcategory: POISubcategoryGrocery},
		"department_store": {Category: POICategoryShopping, Subcategory: POISubcategoryDepartmentStore},
		"bakery":           {Category: POICategoryShopping, Subcategory: POISubcategoryBakery},
		"butcher":          {Category: POICategoryShopping, Subcategory: POISubcategoryButcher},
		"greengrocer":      {Category: POICategoryShopping, Subcategory: POISubcategoryGreengrocer},
	},
	"leisure": {
		"park":          {Category: POICategoryLeisure, Subcategory: POISubcategoryPark},
		"garden":        {Category: POICategoryLeisure, Subcategory: POISubcategoryGarden},
		"playground":    {Category: POICategoryLeisure, Subcategory: POISubcategoryPlayground},
		"sports_centre": {Category: POICategoryLeisure, Subcategory: POISubcategorySportsCentre},
	},
	"tourism": {
		"attraction": {Category: POICategoryLeisure, Subcategory: POISubcategoryAttraction},
		"viewpoint":  {Category: POICategoryLeisure, Subcategory: POISubcategoryViewpoint},
		"museum":     {Category: POICategoryLeisure, Subcategory: POISubcategoryMuseum},
	},
	"historic": {
		"monument": {Category: POICategoryLeisure, Subcategory: POISubcategoryMonument},
		"castle":   {Category: POICategoryLeisure, Subcategory: POISubcategoryCastle},
	},
	"healthcare": {
		// healthcare=* tag - subcategory is the value itself
	},
}

// CategoryMapping represents mapping from OSM tag to category/subcategory
type CategoryMapping struct {
	Category    string
	Subcategory string
}

// ValidPOICategories returns list of valid POI categories
func ValidPOICategories() []string {
	return []string{
		POICategoryHealthcare,
		POICategoryShopping,
		POICategoryEducation,
		POICategoryLeisure,
		POICategoryFoodDrink,
	}
}

// IsValidPOICategory checks if category is valid
func IsValidPOICategory(category string) bool {
	validCategories := ValidPOICategories()
	for _, c := range validCategories {
		if c == category {
			return true
		}
	}
	return false
}
