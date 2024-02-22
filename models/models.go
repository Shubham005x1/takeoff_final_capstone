package models

type GroceryItem struct {
	ID                  int     `json:"id"`
	ProductName         string  `json:"productName" validate:"required"`
	Category            string  `json:"category" validate:"required"`
	Price               float64 `json:"price" validate:"required"`
	Weight              float64 `json:"weight" validate:"required"`
	Vegetarian          bool    `json:"vegetarian"`
	Image               string  `json:"imageURL" validate:"required"`
	Thumbnail           string  `json:"thumbnailURL" validate:"required"`
	Manufacturer        string  `json:"manufacturer" validate:"required"`
	Brand               string  `json:"brand" validate:"required"`
	ItemPackageQuantity int     `json:"itemPackageQuantity" validate:"required"`
	PackageInformation  string  `json:"packageInformation" validate:"required"`
	CountryOfOrigin     string  `json:"countryOfOrigin" validate:"required"`
}
