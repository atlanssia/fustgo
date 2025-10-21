package plugins

// Import all plugins to register them via init() functions

import (
	// Input plugins
	_ "github.com/atlanssia/fustgo/plugins/input/csv"
	
	// Processor plugins
	_ "github.com/atlanssia/fustgo/plugins/processor/filter"
	_ "github.com/atlanssia/fustgo/plugins/processor/mapping"
	
	// Output plugins
	_ "github.com/atlanssia/fustgo/plugins/output/csv"
)

// This file ensures all plugins are imported and registered
// Import this package in main.go to load all plugins
