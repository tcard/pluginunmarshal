// Package pluginunmarshal unmarshals Go plugins into structs.
package pluginunmarshal

import (
	"fmt"
	"plugin"
	"reflect"
	"strings"
)

// Open is a convenience function for calling "plugin".Open and then calling
// Unmarshal with the resulting plugin.
func Open(path string, v interface{}) error {
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}
	return Unmarshal(p, v)
}

// Unmarshal stores exported values from a plugin into the struct pointed to
// by v.
//
// By default, each exported field in the struct will be assigned to a
// package-level exported value in the plugin with the same name as the field.
//
// The plugin value's type must be assignable to the field's type.
//
// If the plugin value is variable, the field's type can also be a pointer to
// the variable's type; in that case, the field will point to the variable.
//
// If no such value exists in the plugin, Unmarshal returns an error.
//
// This behavior can be modified by a struct tag with key "plugin", as follows:
//
//   // Field is ignored by this package.
//   Field int `plugin:"-"`
//
//   // Field is mapped to the package-level value Other from the plugin.
//   Field int `plugin:"Other"`
//
//   // If a value named "Field" isn't exported from the plugin, the struct
//   // field Field will be ignored instead of Unmarshal returning an error.
//   Field int `plugin:",omitempty"`
//
// See package-level examples.
func Unmarshal(p *plugin.Plugin, v interface{}) error {
	rv := reflect.ValueOf(v)
	rt := rv.Type()

	if kind := rt.Kind(); kind == reflect.Ptr {
		rv = rv.Elem()
		rt = rv.Type()
	} else {
		return fmt.Errorf(unmarshalErrFmt, rt, kind)
	}

	if kind := rt.Kind(); kind != reflect.Struct {
		return fmt.Errorf(unmarshalErrFmt, rt, kind)
	}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		name := field.Name
		omitEmpty := false
		if tag, ok := field.Tag.Lookup("plugin"); ok {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
			for i := 1; i < len(parts); i++ {
				switch parts[i] {
				case "omitempty":
					omitEmpty = true
				}
			}
		}

		pv, err := p.Lookup(name)
		if err != nil {
			if !omitEmpty {
				return err
			}
			continue
		}
		fromPlugin := reflect.ValueOf(pv)

		if t := fromPlugin.Type(); !t.AssignableTo(field.Type) {
			if t.Kind() == reflect.Ptr {
				fromPlugin = fromPlugin.Elem()
			}
			if t := fromPlugin.Type(); !t.AssignableTo(field.Type) {
				return fmt.Errorf("pluginunmarshal: value %s of type %v cannot be assigned to field %v of type %v", name, t, field.Name, field.Type)
			}
		}

		rv.Field(i).Set(fromPlugin)
	}

	return nil
}

const unmarshalErrFmt = "pluginunmarshal: can unmarshal to pointer to struct only, got %v of kind %v"
