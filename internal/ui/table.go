package ui

import (
	"bytes"
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// newRecordsTable creates a tablewriter.Table with the shared configuration used
// by the report, log, and stats renderers. headers and footer may be nil.
func newRecordsTable(b *bytes.Buffer, rs reportStyles, headers []string, footer []string) (*tablewriter.Table, error) {
	opts := []tablewriter.Option{
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{
					Alignment:  tw.AlignCenter,
					AutoWrap:   tw.WrapNone,
					AutoFormat: tw.Off,
				},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{
					Alignment: tw.AlignLeft,
					AutoWrap:  tw.WrapNone,
				},
			},
			Footer: tw.CellConfig{
				Formatting: tw.CellFormatting{
					Alignment:  tw.AlignCenter,
					AutoWrap:   tw.WrapNone,
					AutoFormat: tw.Off,
				},
			},
		}),
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{Symbols: rs.symbols(tw.StyleASCII)})),
	}

	if len(headers) > 0 {
		opts = append(opts, tablewriter.WithHeader(headers))
	}

	if len(footer) > 0 {
		opts = append(opts, tablewriter.WithFooter(footer))
	}

	table := tablewriter.NewTable(b, opts...)

	return table, nil
}

// renderRecordsTable builds a table with the given data and returns the rendered string.
func renderRecordsTable(rs reportStyles, headers []string, footer []string, data [][]string) (string, error) {
	b := bytes.Buffer{}

	table, err := newRecordsTable(&b, rs, headers, footer)
	if err != nil {
		return "", err
	}

	if err := table.Bulk(data); err != nil {
		return "", fmt.Errorf("%w: %s", errCouldntAddDataToTable, err.Error())
	}

	if err := table.Render(); err != nil {
		return "", fmt.Errorf("%w: %s", errCouldntRenderTable, err.Error())
	}

	return b.String(), nil
}
