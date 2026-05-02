package properties

import "errors"

func validateServiceIDs(serviceIDs []int32) error {
	for i, serviceID := range serviceIDs {
		if serviceID <= 0 {
			return errors.New("services[" + indexString(i) + "] must be greater than 0")
		}
	}

	return nil
}

func validateClauseInputs(clauses []CreatePropertyClauseInput) error {
	for i, clause := range clauses {
		if clause.ClauseID <= 0 {
			return errors.New("clauses[" + indexString(i) + "].clause_id must be greater than 0")
		}
	}

	return nil
}

func validatePriceInputs(input UpdatePropertyPricesInput) error {
	if input.SalePrice != nil {
		if input.SalePrice.SalePrice <= 0 {
			return errors.New("sale_price.sale_price must be greater than 0")
		}
	}

	for i, rentPrice := range input.RentPrices {
		if rentPrice.PeriodID <= 0 {
			return errors.New("rent_prices[" + indexString(i) + "].period_id must be greater than 0")
		}

		if rentPrice.RentPrice <= 0 {
			return errors.New("rent_prices[" + indexString(i) + "].rent_price must be greater than 0")
		}
	}

	return nil
}
