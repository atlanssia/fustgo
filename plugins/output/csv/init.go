package csv

import (
	"github.com/atlanssia/fustgo/internal/plugin"
)

func init() {
	// Register CSV output plugin
	plugin.RegisterOutput("csv", &CSVOutputPlugin{})
}
