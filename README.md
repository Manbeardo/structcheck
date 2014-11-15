structcheck
===========

Checks that constraints on structs are met. Constraints are read as a comma-delimited list on the 'checks' annotation. Validate constraints by running `structcheck.Validate()`.

See structcheck.Check in godoc for the list of possible constraints.

Example usage:
```golang
package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/Manbeardo/structcheck"
)

type MyJsonObjectType struct {
  NestedObject struct{
    A *int `checks:"NotNil"`
    B *int `checks:"NotNil"`
  }
}

var badJson = []byte(`{
  "NestedObject":{
    "A":1
  }
}`)

func main() {
  var o MyJsonObjectType
  json.NewDecoder(bytes.NewBuffer(badJson)).Decode(&o)
  err := structcheck.Validate(o)
  if err != nil {
    fmt.Println(err.Error())
  }
}
```
Prints:
```
The following field(s) failed checks: 
    MyJsonObjectType.NestedObject.B: NotNil: (*int)(nil)
```
