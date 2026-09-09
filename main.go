package main

import (
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/cmd"
)

func main() {
	log.SetPrefix("")
	log.SetReportTimestamp(false)
	cmd.Execute()
}
