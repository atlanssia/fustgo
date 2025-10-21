package filter

import (
	"github.com/atlanssia/fustgo/internal/plugin"
)

func init() {
	plugin.RegisterProcessor("filter", &FilterProcessor{})
}
