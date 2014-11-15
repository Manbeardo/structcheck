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

func (v metaValue) getChecks() map[Check]interface{} {
	checks := make(map[Check]interface{}, 0)
	if v.tag != nil {
		for _, str := range strings.Split(v.tag.Get("checks"), ",") {
			if check, ok := str2check[strings.ToLower(str)]; ok {
				checks[check] = nil
			}
		}
	}
	return checks
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
