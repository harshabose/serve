package interceptor

import (
	"errors"
	"strings"
)

func flattenErrs(errs []error) error {
	var errs2 []error
	for _, e := range errs {
		if e != nil {
			errs2 = append(errs2, e)
		}
	}
	if len(errs2) == 0 {
		return nil
	}
	return multiError(errs2)
}

type multiError []error

func (errs multiError) Error() string {
	var errStrings []string

	for _, err := range errs {
		if err != nil {
			errStrings = append(errStrings, err.Error())
		}
	}

	if len(errStrings) == 0 {
		return "multiError must contain multiple error but is empty"
	}

	return strings.Join(errStrings, "\n")
}

func (errs multiError) Is(err error) bool {
	for _, e := range errs {
		if errors.Is(e, err) {
			return true
		}
		var errs2 multiError
		if errors.As(e, &errs2) {
			if errs2.Is(err) {
				return true
			}
		}
	}
	return false
}
