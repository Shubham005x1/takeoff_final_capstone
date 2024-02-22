package validations

import (
	"fmt"
	"strconv"
)

func ValidatePrice(priceStr string) (float64, error) {
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Price cannot be a character value. Please enter a valid value")
	}
	if price < 0 {
		return 0, fmt.Errorf("Price should not be a non-negative value")
	}
	return price, nil
}
func ValidateItemPackageQuantity(itemPackageQuantityStr string) (int, error) {
	itemPackageQuantity, err := strconv.Atoi(itemPackageQuantityStr)
	if err != nil {
		return 0, fmt.Errorf("ItemPackageQuantity Cannot be Character value, Please Enter Valid value")
	}

	if itemPackageQuantity < 0 {
		return 0, fmt.Errorf("ItemPackageQuantity Should Not be a non-negative value")
	}

	return itemPackageQuantity, nil
}
