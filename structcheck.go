package structcheck

import (
	"container/list"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// value plus information from a few levels up
type metaValue struct {
	reflect.Value
	Name   []string
	Number []int
	tag    *reflect.StructTag
}

func (v metaValue) buildDeeperName(n string) []string {
	name := make([]string, len(v.Name), len(v.Name)+1)
	copy(name, v.Name)
	return append(name, n)
}

func (v metaValue) buildDeeperNumber(n int) []int {
	num := make([]int, len(v.Number), len(v.Number)+1)
	copy(num, v.Number)
	return append(num, n)
}

func (v metaValue) Field(i int) metaValue {
	f := v.Value.Type().Field(i)
	return metaValue{
		Value:  v.Value.Field(i),
		Name:   v.buildDeeperName(f.Name),
		Number: v.buildDeeperNumber(i),
		tag:    &f.Tag,
	}
}

// InterfaceValue returns the Value wrapped by v (assuming v is an interface)
func (v metaValue) InterfaceValue() metaValue {
	v2 := reflect.ValueOf(v.Value.Interface())
	n := v.buildDeeperName(fmt.Sprintf("(%v)", v2.Type().Name()))
	return metaValue{Value: v2, Name: n, tag: v.tag, Number: v.Number}
}

func (v metaValue) Indirect() metaValue {
	return metaValue{Value: reflect.Indirect(v.Value), Name: v.Name, tag: v.tag, Number: v.Number}
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

// drills down (follows pointer and interface indirection) to a struct and recursively runs checks on all fields.
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
		// check NotNil conditions
		switch v.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
			if v.IsNil() && v.getChecks().NotNil {
				f := newField(v)
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
	Name   string
	Value  string
	Number string
}

func newField(v metaValue) Field {
	n := make([]string, len(v.Number))
	for i, num := range v.Number {
		n[i] = strconv.Itoa(num)
	}
	return Field{
		Name:   strings.Join(v.Name, "."),
		Value:  fmt.Sprintf("%#v", v.Value.Interface()),
		Number: strings.Join(n, "."),
	}
}

type ByFieldOrder []Field

func (a ByFieldOrder) Len() int {
	return len(a)
}

func (a ByFieldOrder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByFieldOrder) Less(i, j int) bool {
	n1 := strings.Split(a[i].Number, ".")
	n2 := strings.Split(a[j].Number, ".")
	minLen := int(math.Min(float64(len(n1)), float64(len(n2))))
	for k := 0; k < minLen; k++ {
		if n1[k] != n2[k] {
			num1, _ := strconv.Atoi(n1[k])
			num2, _ := strconv.Atoi(n2[k])
			return num1 < num2
		}
	}
	return len(n1) < len(n2)
}
