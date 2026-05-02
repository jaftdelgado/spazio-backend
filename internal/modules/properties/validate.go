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
