package symbol

import (
	"reflect"

	"github.com/barab-i/incipio/internal/theme"
)

// Symbols contains the map of symbols for packages used by plugins,
// making them available to Yaegi interpreters.
var Symbols = map[string]map[string]reflect.Value{
	// Expose theme variable for Yaegi plugins
	"github.com/barab-i/incipio/internal/theme": {
		"CurrentTheme": reflect.ValueOf(&theme.CurrentTheme).Elem(),
	},
}
