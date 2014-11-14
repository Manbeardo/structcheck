package structcheck

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type CycleNode struct {
	*CycleNode
}

func TestSelfReference(t *testing.T) {
	node := new(CycleNode)
	node.CycleNode = node
	assert.Nil(t, Validate(node))
}

func TestSmallCycle(t *testing.T) {
	n1 := new(CycleNode)
	n2 := &CycleNode{CycleNode: n1}
	n1.CycleNode = n2
	assert.Nil(t, Validate(n1))
}

type SlicyStruct struct {
	Nilly   []interface{}
	NoNilly []interface{} `checks:"NotNil"`
}

func TestGoodSlicyStruct(t *testing.T) {
	assert.Nil(t, Validate(SlicyStruct{
		Nilly:   nil,
		NoNilly: make([]interface{}, 0),
	}))
}

func TestBadSlicyStruct(t *testing.T) {
	assert.Error(t, Validate(SlicyStruct{
		Nilly:   nil,
		NoNilly: nil,
	}))
}

type BigStruct struct {
	Slicy SlicyStruct
}

func TestNilOneLayerDown(t *testing.T) {
	err := Validate(BigStruct{
		Slicy: SlicyStruct{
			Nilly:   nil,
			NoNilly: nil,
		},
	})
	assert.Error(t, err)
	assert.NoError(t, checkDeepEqual([][]string{[]string{"BigStruct", "Slicy", "NoNilly"}}, err.(ErrorNilField).FieldNames))
}

func checkDeepEqual(e, r interface{}) error {
	if reflect.DeepEqual(e, r) {
		return nil
	} else {
		return notEqualError{e: e, r: r}
	}
}

type notEqualError struct {
	e, r interface{}
}

func (e notEqualError) Error() string {
	return fmt.Sprintf("Not equal: %+v != %+v", e.e, e.r)
}