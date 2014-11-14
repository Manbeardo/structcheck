package structcheck

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
)

// value plus information from a few levels up
type metaValue struct {
	reflect.Value
	Name []string
	tag  *reflect.StructTag
}

func (v metaValue) buildDeeperName(n string) []string {
	name := make([]string, len(v.Name), len(v.Name)+1)
	copy(name, v.Name)
	return append(name, n)
}

func (v metaValue) Field(i int) metaValue {
	v2 := v.Value.Field(i)
	f := v.Value.Type().Field(i)
	n := v.buildDeeperName(f.Name)
	return metaValue{Value: v2, Name: n, tag: &f.Tag}
}

// InterfaceValue returns the Value wrapped by v (assuming v is an interface)
func (v metaValue) InterfaceValue() metaValue {
	v2 := reflect.ValueOf(v.Value.Interface())
	n := v.buildDeeperName(fmt.Sprintf("(%v)", v2.Type().Name()))
	return metaValue{Value: v2, Name: n, tag: v.tag}
}

func (v metaValue) Indirect() metaValue {
	return metaValue{Value: reflect.Indirect(v.Value), Name: v.Name, tag: v.tag}
}

func (v metaValue) getChecks() checks {
	var c checks
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

type checks struct {
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

func (q *valueQueue) Push(v metaValue) {
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

func (q *valueQueue) Pop() metaValue {
	front := q.queue.Front()
	v := front.Value.(metaValue)
	q.queue.Remove(front)
	return v
}

func (q *valueQueue) Len() int {
	return q.queue.Len()
}

// recursively runs checks on all
func Validate(o interface{}) error {
	if o == nil {
		return ErrorNilValue{}
	}
	top := reflect.ValueOf(o)
	// drill down to first struct
	for k := top.Kind(); k == reflect.Ptr || k == reflect.Interface; k = top.Kind() {
		if top.IsNil() {
			return ErrorNilValue{}
		} else if k == reflect.Ptr {
			top = reflect.Indirect(top)
		} else if k == reflect.Interface {
			top = reflect.ValueOf(top.Interface())
		}
	}

	if top.Kind() != reflect.Struct {
		return ErrorInvalidKind{Kind: top.Kind()}
	}
	// Breadth first search to find nil required fields
	namedTop := metaValue{Value: top, Name: []string{top.Type().Name()}}
	field2checks := make(map[Field][]Check)
	q := newValueQueue()
	q.Push(namedTop)
	for q.Len() > 0 {
		v := q.Pop()
		// check NotNil conditions
		switch v.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
			if v.IsNil() && v.getChecks().NotNil {
				f := NewField(v)
				field2checks[f] = append(field2checks[f], NotNil)
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

	if len(field2checks) != 0 {
		return ErrorChecksFailed{Field2Checks: field2checks}
	} else {
		return nil
	}
}

type Check string

const (
	NotNil Check = "NotNil"
)

type Field struct {
	Name  string
	Value string
}

func NewField(v metaValue) Field {
	return Field{Name: strings.Join(v.Name, "."), Value: fmt.Sprintf("%#v", v.Value.Interface())}
}

// returned when the top level object does not drill down to a struct.
type ErrorInvalidKind struct {
	reflect.Kind
}

func (e ErrorInvalidKind) Error() string {
	return fmt.Sprintf("Provided object must drill down to a struct. Received: %v", e.Kind)
}

// returned when a top level nil is received
type ErrorNilValue struct{}

func (e ErrorNilValue) Error() string {
	return fmt.Sprintf("Provided object must drill down to a struct. Encountered nil.")
}

// returned when checks fail on fields
type ErrorChecksFailed struct {
	Field2Checks map[Field][]Check
}

func (e ErrorChecksFailed) Error() string {
	failLines := make([]string, 0, len(e.Field2Checks))
	for field, checks := range e.Field2Checks {
		fails := make([]string, 0, len(checks))
		for _, check := range checks {
			fails = append(fails, string(check))
		}
		failLines = append(failLines, fmt.Sprintf("%v: %v: %v", strings.Join(fails, ", "), field.Name, field.Value))
	}
	return fmt.Sprintf("The following field(s) failed checks: \n\t%v", strings.Join(failLines, "\n\t"))
}
