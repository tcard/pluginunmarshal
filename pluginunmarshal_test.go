package pluginunmarshal_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"testing"

	"github.com/tcard/pluginunmarshal"
)

func Example() {
	// Assume pathToPlugin refers to a Go plugin file compiled from this code:
	//
	//   package main
	//
	//   var Hello = "Hello from a plugin!"
	//
	//   func Add(a, b int) int {
	//   	return a + b
	//   }
	//

	var v struct {
		Add     func(a, b int) int
		MyHello string `plugin:"Hello"`
		Ignored bool   `plugin:"-"`
	}

	err := pluginunmarshal.Open(pathToPlugin, &v)
	if err != nil {
		panic(err)
	}

	fmt.Println(v.Add(2, 3))
	fmt.Println(v.MyHello)
	// Output:
	// 5
	// Hello from a plugin!
}

func TestUnmarshal(t *testing.T) {
	p, err := plugin.Open(pathToPlugin)
	if err != nil {
		t.Fatalf("%v", err)
	}

	for _, testCase := range []struct {
		name   string
		dst    interface{}
		errMsg string
		do     func(*testing.T, reflect.Value)
	}{
		{
			name: "basic OK",
			dst: &struct {
				Hello string
				Add   func(a, b int) int
			}{},
			do: func(t *testing.T, dst reflect.Value) {
				if expected, got := "Hello from a plugin!", dst.FieldByName("Hello").String(); expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}
				if expected, got := 5, dst.FieldByName("Add").Call([]reflect.Value{
					reflect.ValueOf(2),
					reflect.ValueOf(3),
				})[0].Interface().(int); expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}
			},
		},

		{
			name: "Add failing because of type",
			dst: &struct {
				Hello string
				Add   string
			}{},
			errMsg: "pluginunmarshal: value Add of type func(int, int) int cannot be assigned to field Add of type string",
		},

		{
			name: "failing because of unexisting field",
			dst: &struct {
				Hello       string
				NotExisting string
			}{},
			errMsg: "plugin: symbol NotExisting not found in plugin",
		},

		{
			name: "ignore unexisting field, OK",
			dst: &struct {
				Hello       string
				NotExisting string `plugin:"-"`
			}{},
		},

		{
			name: "failing because non-ptr dst",
			dst: struct {
				Hello string
			}{},
			errMsg: "pluginunmarshal: can unmarshal to pointer to struct only",
		},

		{
			name:   "failing because non-struct dst",
			dst:    new(int),
			errMsg: "pluginunmarshal: can unmarshal to pointer to struct only",
		},

		{
			name: "name mapping",
			dst: &struct {
				MyHello string `plugin:"Hello"`
			}{},
			do: func(t *testing.T, dst reflect.Value) {
				if expected, got := "Hello from a plugin!", dst.FieldByName("MyHello").String(); expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}
			},
		},

		{
			name: "omitempty",
			dst: &struct {
				MyHello     string `plugin:"Hello,omitempty"`
				NotExisting string `plugin:"NotExisting,omitempty"`
			}{},
			do: func(t *testing.T, dst reflect.Value) {
				if expected, got := "Hello from a plugin!", dst.FieldByName("MyHello").String(); expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}
				if expected, got := "", dst.FieldByName("NotExisting").String(); expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}
			},
		},

		{
			name: "omitempty but wrong type still fails",
			dst: &struct {
				MyHello int `plugin:"Hello,omitempty"`
			}{},
			errMsg: "pluginunmarshal: value Hello of type string cannot be assigned to field MyHello of type int",
		},

		{
			name: "pointer points to plugin var",
			dst: &struct {
				MyHello *string `plugin:"Hello,omitempty"`
			}{},
			do: func(t *testing.T, dst reflect.Value) {
				ptr := dst.FieldByName("MyHello").Interface().(*string)
				if expected, got := "Hello from a plugin!", *ptr; expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}

				fromPlugin, err := p.Lookup("Hello")
				if err != nil {
					t.Errorf("%v", err)
					return
				}

				if expected, got := ptr, fromPlugin.(*string); expected != got {
					t.Errorf("expected %q, got %q", expected, got)
				}
			},
		},
	} {
		t.Run("testCase="+testCase.name, func(t *testing.T) {
			dst := testCase.dst
			err := pluginunmarshal.Unmarshal(p, dst)
			if err != nil {
				if testCase.errMsg != "" {
					if expected, got := testCase.errMsg, err.Error(); !strings.HasPrefix(got, expected) {
						t.Errorf("expected error %q, got %q", expected, got)
					}
				} else {
					t.Errorf("unexpected error: %v", err)
				}
				return
			} else if testCase.errMsg != "" {
				t.Errorf("expected error %q, got no error", testCase.errMsg)
				return
			}

			if testCase.do != nil {
				testCase.do(t, reflect.ValueOf(dst).Elem())
			}
		})
	}
}

var pathToPlugin string

const examplePlugin = `
package main

var Hello = "Hello from a plugin!"

func Add(a, b int) int {
	return a + b
}
`

func init() {
	goCmdPath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}

	dirPath, err := ioutil.TempDir("", "pluginunmarshal_test")
	if err != nil {
		panic(err)
	}
	srcPath := filepath.Join(dirPath, "exampleplugin.go")

	err = ioutil.WriteFile(srcPath, []byte(examplePlugin), 0777)
	if err != nil {
		panic(err)
	}

	pathToPlugin = filepath.Join(dirPath, "exampleplugin")

	output, err := exec.Command(goCmdPath, "build", "-buildmode=plugin", "-o", pathToPlugin, srcPath).CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("output: %s, err: %v", string(output), err))
	}
}
