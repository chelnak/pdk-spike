package utils

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

type TableOptions struct {
	Header []string
	Lines  [][]string
}

// RenderTable renders a table with specified options
func RenderTable(opts TableOptions) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(opts.Header)
	table.AppendBulk(opts.Lines)
	table.Render()
}
