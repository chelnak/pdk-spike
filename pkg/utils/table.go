package utils

import (
	"github.com/olekukonko/tablewriter"
	"os"
)

type TableOptions struct {
	Header []string
	Lines  [][]string
}

func RenderTable(opts TableOptions) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(opts.Header)
	table.AppendBulk(opts.Lines)
	table.Render()
}
