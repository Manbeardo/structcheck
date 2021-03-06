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
	assert.NoError(t, Validate(node))
}

func TestSmallCycle(t *testing.T) {
	n1 := new(CycleNode)
	n2 := &CycleNode{CycleNode: n1}
	n1.CycleNode = n2
	assert.NoError(t, Validate(n1))
}

type SlicyStruct struct {
	Nilly   []interface{}
	NoNilly []interface{} `checks:"NotNil"`
}

func TestGoodSlicyStruct(t *testing.T) {
	assert.NoError(t, Validate(SlicyStruct{
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
	err = checkDeepEqual(map[Field][]string{Field{Name: "BigStruct.Slicy.NoNilly", Value: "[]interface {}(nil)", Number: "0.1"}: []string{"NotNil"}}, err.(ErrorChecksFailed).Field2Checks)
	assert.NoError(t, err)
}

func TestTopLevelSlice(t *testing.T) {
	err := Validate([]string{})
	assert.Error(t, err)
	assert.IsType(t, ErrorInvalidKind{}, err)
}

func TestSecondLevelSlice(t *testing.T) {
	err := Validate(&[]string{})
	assert.Error(t, err)
	assert.IsType(t, ErrorInvalidKind{}, err)
}

func TestTopLevelNil(t *testing.T) {
	err := Validate(nil)
	assert.Error(t, err)
	assert.IsType(t, ErrorNilValue{}, err)
}

func TestSecondLevelNil(t *testing.T) {
	err := Validate(new(*struct{}))
	assert.Error(t, err)
	assert.IsType(t, ErrorNilValue{}, err)
}

func TestNilIntPointer(t *testing.T) {
	err := Validate(struct {
		IntPtr *int `checks:"NotNil"`
	}{IntPtr: nil})
	assert.Error(t, err)
	assert.IsType(t, ErrorChecksFailed{}, err)
}

// this is an eyeball test for the terminal output
func TestVeryLongFieldNameStruct(t *testing.T) {
	err := Validate(struct {
		VeryVeryVeryVeryVeryVeryLongFieldName *int `checks:"NotNil"`
		B                                     *int `checks:"NotNil"`
		StructWithManyMembers                 struct {
			A *int  `checks:"NotNil"`
			B *int  `checks:"NotNil"`
			C *int  `checks:"NotNil"`
			D *int  `checks:"NotNil"`
			E *int  `checks:"NotNil"`
			F []int `checks:"NotEmpty"`
		}
	}{})
	assert.Error(t, err)
	fmt.Println(err.Error())
}

func TestAllSuccessCases(t *testing.T) {
	err := Validate(struct {
		A *struct{} `checks:"NotNil"`
		B *int      `checks:"Nil"`
		C int       `checks:"Positive"`
		D int       `checks:"Negative"`
		E int       `checks:"NoSign"`
		F []int     `checks:"NotEmpty"`
		G []int     `checks:"Empty"`
	}{
		A: &struct{}{},
		B: nil,
		C: 1,
		D: -1,
		E: 0,
		F: []int{0},
		G: []int{},
	})
	assert.NoError(t, err)
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
	return fmt.Sprintf("Not equal: %#v != %#v", e.e, e.r)
}
