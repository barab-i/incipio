// Code generated by 'yaegi extract github.com/charmbracelet/bubbles/viewport'. DO NOT EDIT.

package symbol

import (
	"github.com/charmbracelet/bubbles/viewport"
	"reflect"
)

func init() {
	Symbols["github.com/charmbracelet/bubbles/viewport/viewport"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"DefaultKeyMap": reflect.ValueOf(viewport.DefaultKeyMap),
		"New":           reflect.ValueOf(viewport.New),
		"Sync":          reflect.ValueOf(viewport.Sync),
		"ViewDown":      reflect.ValueOf(viewport.ViewDown),
		"ViewUp":        reflect.ValueOf(viewport.ViewUp),

		// type definitions
		"KeyMap": reflect.ValueOf((*viewport.KeyMap)(nil)),
		"Model":  reflect.ValueOf((*viewport.Model)(nil)),
	}
}
