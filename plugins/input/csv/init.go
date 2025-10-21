package csv

import (
	"github.com/atlanssia/fustgo/internal/plugin"
)

func init() {
	// Register CSV input plugin
	plugin.RegisterInput("csv", &CSVInputPlugin{})
}
