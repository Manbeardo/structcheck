package structcheck

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
)

type namedValue struct {
	reflect.Value
	Name []string
	tag  *reflect.StructTag
}

func (v namedValue) deeperName(n string) []string {
	name := make([]string, len(v.Name), len(v.Name)+1)
	copy(name, v.Name)
	return append(name, n)
}

func (v namedValue) Field(i int) namedValue {
	v2 := v.Value.Field(i)
	f := v.Value.Type().Field(i)
	n := v.deeperName(f.Name)
	return namedValue{Value: v2, Name: n, tag: &f.Tag}
}

// InterfaceValue returns the Value wrapped by v (assuming v is an interface)
func (v namedValue) InterfaceValue() namedValue {
	v2 := reflect.ValueOf(v.Value.Interface())
	n := v.deeperName(fmt.Sprintf("(%v)", v2.Type().Name()))
	return namedValue{Value: v2, Name: n, tag: v.tag}
}

func (v namedValue) Indirect() namedValue {
	return namedValue{Value: reflect.Indirect(v.Value), Name: v.Name, tag: v.tag}
}

func (v namedValue) GetChecks() Checks {
	var c Checks
	if v.tag != nil {
		for _, str := range strings.Split(v.tag.Get("checks"), ",") {
			switch strings.ToLower(str) {
			case "notnil":
				c.NotNil = true
			}
		}
	}
	return c
}

type Checks struct {
	NotNil bool
}

// Breadth First Search queue for reflective struct exploration. Prevents infinite recursion by marking pointers and refusing to push marked pointers.
type valueQueue struct {
	queuedPointers map[uintptr]interface{}
	queue          *list.List
}

func newValueQueue() *valueQueue {
	return &valueQueue{
		queuedPointers: make(map[uintptr]interface{}),
		queue:          list.New(),
	}
}

func (q *valueQueue) Push(v namedValue) {
	kind := v.Kind()
	// take internal value of interfaces
	if kind == reflect.Interface {
		v = v.InterfaceValue()
	}
	// mark pointers
	if kind == reflect.Ptr && !v.IsNil() {
		ptr := v.Pointer()
		if _, present := q.queuedPointers[ptr]; present {
			return
		} else {
			q.queuedPointers[ptr] = nil
		}
	}
	// enqueue value
	q.queue.PushBack(v)
	return
}

func (q *valueQueue) Pop() namedValue {
	front := q.queue.Front()
	v := front.Value.(namedValue)
	q.queue.Remove(front)
	return v
}

func (q *valueQueue) Len() int {
	return q.queue.Len()
}

// recursively checks that all required fields of struct o are non-nil. Not thread-safe.
func Validate(o interface{}) error {
	top := reflect.ValueOf(o)
	// drill down to first struct
	for k := top.Kind(); k == reflect.Ptr || k == reflect.Interface; k = top.Kind() {
		if top.IsNil() {
			return fmt.Errorf("o must be non-nil")
		} else if k == reflect.Ptr {
			top = reflect.Indirect(top)
		} else if k == reflect.Interface {
			top = reflect.ValueOf(top.Interface())
		}
	}

	if top.Kind() != reflect.Struct {
		return fmt.Errorf("o must be a struct type. Received: %v", top.Kind())
	}
	// Breadth first search to find nil required fields
	namedTop := namedValue{Value: top, Name: []string{top.Type().Name()}}
	badFields := make([][]string, 0)
	q := newValueQueue()
	q.Push(namedTop)
	for q.Len() > 0 {
		v := q.Pop()
		// check NotNil conditions
		switch v.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
			if v.IsNil() && v.GetChecks().NotNil {
				badFields = append(badFields, v.Name)
			}
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

	if len(badFields) != 0 {
		return ErrorNilField{FieldNames: badFields}
	} else {
		return nil
	}
}

type ErrorNilField struct {
	FieldNames [][]string
}

func (e ErrorNilField) Error() string {
	joinedNames := make([]string, len(e.FieldNames))
	for _, n := range e.FieldNames {
		joinedNames = append(joinedNames, strings.Join(n, "."))
	}
	return fmt.Sprintf("The following required field(s) were nil: \n\t%v\n", strings.Join(joinedNames, "\n\t"))
}
