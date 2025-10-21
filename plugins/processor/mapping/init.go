package mapping

import (
	"github.com/atlanssia/fustgo/internal/plugin"
)

func init() {
	plugin.RegisterProcessor("mapping", &MappingProcessor{})
}
