// Command drillback verifies that a backup would actually come back.
//
// This file is wiring only. Everything it needs lives in internal/, so that a future
// second entry point, or drillback used as a library, costs nothing.
package main

import (
	"os"

	"github.com/spelingbee/drillback/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
