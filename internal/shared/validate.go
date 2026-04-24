package shared

import "errors"

type ValidationRule struct {
	Fail bool
	Msg  string
}

func Validate(rules []ValidationRule) error {
	for _, r := range rules {
		if r.Fail {
			return errors.New(r.Msg)
		}
	}
	return nil
}
