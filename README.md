# pluginunmarshal [![Build Status](https://secure.travis-ci.org/tcard/pluginunmarshal.svg?branch=master)](http://travis-ci.org/tcard/pluginunmarshal) [![GoDoc](https://godoc.org/github.com/tcard/pluginunmarshal?status.svg)](https://godoc.org/github.com/tcard/pluginunmarshal)

Package pluginunmarshal unmarshals Go plugins into structs.

```go
// Assume pathToPlugin refers to a Go plugin file compiled from this code:
//
//   package main
//
//   var Hello = "Hello from a plugin!"
//
//   func Add(a, b int) int {
//      return a + b
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
```

[See the docs](https://godoc.org/github.com/tcard/pluginunmarshal) for more.
