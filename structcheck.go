package structcheck

import (
	"reflect"
)

func drillDown(v reflect.Value) (reflect.Value, error) {
	for k := v.Kind(); k == reflect.Ptr || k == reflect.Interface; k = v.Kind() {
		if v.IsNil() {
			return v, ErrorNilValue{}
		} else if k == reflect.Ptr {
			v = reflect.Indirect(v)
		} else if k == reflect.Interface {
			v = reflect.ValueOf(v.Interface())
		}
	}
	return v, nil
}

func runChecks(v metaValue) ([]string, error) {
	failedChecks := []string{}
	checks, checkNames, err := v.getChecks()
	if err != nil {
		return nil, err
	}
	for i, check := range checks {
		if !check(v.Value) {
			failedChecks = append(failedChecks, checkNames[i])
		}
	}
	return failedChecks, nil
}

// drills down (follows pointer and interface indirection) to a struct and recursively runs checks on all fields.
func Validate(i interface{}) error {
	return CustomValidate(i, BuildTagCheckFinder(DefaultChecks))
}

// runs Validate with a custom set of checks
func CustomValidate(i interface{}, checkFinder CheckFinder) error {
	// find root node
	if i == nil {
		return ErrorNilValue{}
	}
	top := reflect.ValueOf(i)
	top, err := drillDown(top)
	if err != nil {
		return err
	}

	if top.Kind() != reflect.Struct {
		return ErrorInvalidKind{Type: top.Type()}
	}

	// Breadth first search
	name := top.Type().Name()
	if name == "" {
		name = "(anonymous struct)"
	}
	namedTop := metaValue{
		Value:       top,
		Name:        []string{name},
		CheckFinder: checkFinder,
	}
	field2checks := make(map[Field][]string)
	q := newValueQueue()
	q.Push(namedTop)
	for q.Len() > 0 {
		v := q.Pop()
		failedChecks, err := runChecks(v)
		if err != nil {
			return err
		}
		if len(failedChecks) != 0 {
			field2checks[newField(v)] = failedChecks
		}
		// push new nodes onto queue
		switch v.Kind() {
		case reflect.Ptr:
			if !v.IsNil() {
				q.Push(v.Indirect())
			}
		case reflect.Interface:
			if !v.IsNil() {
				q.Push(v.InterfaceValue())
			}
		case reflect.Struct:
			for j := 0; j < v.NumField(); j++ {
				q.Push(v.Field(j))
			}
		}
	}

	if len(field2checks) != 0 {
		return ErrorChecksFailed{Field2Checks: field2checks}
	} else {
		return nil
	}
}
