package structcheck

import (
	"fmt"
)

func CheckNoNils(i interface{}) error {
	finder, err := BuildFixedCheckFinder([]string{"NotNil"}, DefaultChecks)
	if err != nil {
		panic(fmt.Errorf("Internal error (did you modify DefaultChecks?): %v", err.Error()))
	}
	return CustomValidate(i, DefaultChecks, finder)
}
