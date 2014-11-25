package structcheck

import (
	"fmt"
	"reflect"
	"strings"
)

// Builds a CheckFinder that returns the same set of checks for all fields. checkSet is read at build-time, not check-time.
func BuildFixedCheckFinder(checkNames []string, checkSet map[string]Check) (CheckFinder, error) {
	checks := make([]Check, len(checkNames))
	for i, checkName := range checkNames {
		check, ok := checkSet[checkName]
		if !ok {
			return nil, fmt.Errorf("No check found with name: %v", checkName)
		}
		checks[i] = check
	}
	return func(v metaValue) ([]Check, []string, error) {
		return checks, checkNames, nil
	}, nil
}

// Builds a CheckFinder that runs the named checks on the named fields. checkSet and field2checks are copied and verified at build-time
func BuildStringyCheckFinder(field2checks map[string][]string, checkSet map[string]Check) (CheckFinder, error) {
	f2c := make(map[string][]string, len(field2checks))
	cs := make(map[string]Check, len(checkSet))
	for k, v := range field2checks {
		f2c[k] = v
	}
	for k, v := range checkSet {
		cs[k] = v
	}

	for _, checkNames := range f2c {
		for _, checkName := range checkNames {
			if _, ok := cs[checkName]; !ok {
				return nil, fmt.Errorf("No check found with name: %v", checkName)
			}
		}
	}
	return func(v metaValue) ([]Check, []string, error) {
		name := strings.Join(v.Name[1:], ".")
		if checkNames, ok := f2c[name]; ok {
			checks := make([]Check, len(checkNames))
			for i, checkName := range checkNames {
				checks[i] = cs[checkName]
			}
			return checks, checkNames, nil
		} else {
			return []Check{}, []string{}, nil
		}
	}, nil
}

func CheckFieldExists(i interface{}, fieldName string) bool {
	fields := strings.Split(fieldName, ".")
	v := reflect.ValueOf(i)
	for len(fields) != 0 {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				v = reflect.Zero(v.Type().Elem())
			} else {
				v = v.Elem()
			}
		case reflect.Interface:
			if v.IsNil() {
				return false
			} else {
				v = v.Elem()
			}
		case reflect.Struct:
			v = v.FieldByName(fields[0])
			if v.Kind() == reflect.Invalid {
				return false
			} else {
				// pop
				fields = fields[1:]
			}
		default:
			return false
		}
	}
	return true
}

func CheckFieldsExist(i interface{}, fieldNames []string) error {
	missingFields := []string{}
	for _, name := range fieldNames {
		if CheckFieldExists(i, name) != true {
			missingFields = append(missingFields, name)
		}
	}
	if len(missingFields) != 0 {
		return fmt.Errorf("Field(s) %v do not exist in %#v", missingFields, i)
	} else {
		return nil
	}
}

// checks that no fields in the struct are nil
func CheckNoNils(i interface{}) error {
	finder, err := BuildFixedCheckFinder([]string{"NotNil"}, DefaultChecks)
	if err != nil {
		panic(fmt.Errorf("Internal error (did you modify DefaultChecks?): %v", err.Error()))
	}
	return CustomValidate(i, finder)
}

// checks that the named fields and their parents exist and are not null
func CheckFieldsNotNil(i interface{}, fieldNames []string) error {
	if err := CheckFieldsExist(i, fieldNames); err != nil {
		return err
	}
	field2checks := make(map[string][]string, len(fieldNames))
	checks := []string{"NotNil"}
	for _, name := range fieldNames {
		exploded := strings.Split(name, ".")
		for i := 0; i < len(exploded); i++ {
			subname := strings.Join(exploded[:len(exploded)-i], ".")
			field2checks[subname] = checks
		}
	}

	if finder, err := BuildStringyCheckFinder(field2checks, DefaultChecks); err != nil {
		panic(fmt.Errorf("Internal error (did you modify DefaultChecks?): %v", err.Error()))
	} else {
		return CustomValidate(i, finder)
	}
}
