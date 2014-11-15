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

func runChecks(v metaValue) ([]Check, error) {
	failedChecks := []Check{}
	for check, _ := range v.getChecks() {
		err := check2checker[check](v)
		switch err.(type) {
		case errorCheckFailed:
			failedChecks = append(failedChecks, check)
		default:
			return nil, err
		}
	}
	return failedChecks, nil
}

// drills down (follows pointer and interface indirection) to a struct and recursively runs checks on all fields.
func Validate(o interface{}) error {
	// find root node
	if o == nil {
		return ErrorNilValue{}
	}
	top := reflect.ValueOf(o)
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
	namedTop := metaValue{Value: top, Name: []string{name}}
	field2checks := make(map[Field][]Check)
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
			for i := 0; i < v.NumField(); i++ {
				q.Push(v.Field(i))
			}
		}
	}

	if len(field2checks) != 0 {
		return ErrorChecksFailed{Field2Checks: field2checks}
	} else {
		return nil
	}
}
