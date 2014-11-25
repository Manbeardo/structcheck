package structcheck

import (
	"container/list"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type CheckFinder func(v metaValue) ([]Check, []string, error)

func BuildTagCheckFinder(checkSet map[string]Check) CheckFinder {
	return func(v metaValue) ([]Check, []string, error) {
		return tagCheckFinder(v, checkSet)
	}
}

// Searches struct field tags for check directives
func tagCheckFinder(v metaValue, checkSet map[string]Check) ([]Check, []string, error) {
	checks := []Check{}
	checkNames := []string{}
	if v.tag != nil {
		for _, str := range strings.Split(v.tag.Get("checks"), ",") {
			if str == "" {
				continue
			}
			check, ok := checkSet[str]
			if ok {
				checks = append(checks, check)
				checkNames = append(checkNames, str)
			} else {
				return nil, nil, ErrorIllegalCheck{
					value:  v,
					Reason: fmt.Sprintf("'%v' is not a recognized check type", str),
				}
			}
		}
	}
	return checks, checkNames, nil
}

// value plus information from a few levels up
type metaValue struct {
	reflect.Value
	Name   []string
	Number []int
	CheckFinder
	tag *reflect.StructTag
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
		Value:       v.Value.Field(i),
		Name:        v.buildDeeperName(f.Name),
		Number:      v.buildDeeperNumber(i),
		CheckFinder: v.CheckFinder,
		tag:         &f.Tag,
	}
}

// InterfaceValue returns the Value wrapped by v (assuming v is an interface)
func (v metaValue) InterfaceValue() metaValue {
	v2 := reflect.ValueOf(v.Value.Interface())
	n := v.buildDeeperName(fmt.Sprintf("(%v)", v2.Type().Name()))
	return metaValue{
		Value:       v2,
		Name:        n,
		Number:      v.Number,
		CheckFinder: v.CheckFinder,
		tag:         v.tag,
	}
}

func (v metaValue) Indirect() metaValue {
	return metaValue{
		Value:       reflect.Indirect(v.Value),
		Name:        v.Name,
		Number:      v.Number,
		CheckFinder: v.CheckFinder,
		tag:         v.tag,
	}
}

func (v metaValue) getChecks() ([]Check, []string, error) {
	return v.CheckFinder(v)
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

type Field struct {
	Name   string // qualified field name (e.g. RootType.Field1.Field2)
	Value  string // stringified field value
	Number string // the index in the field tree (e.g. 0.1 for the second field of the first field of the root)
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

// sorts fields by their natural (in-struct) order
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
