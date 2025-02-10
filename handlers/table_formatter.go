package handlers

import (
	"fmt"
	"strings"
)

type TableRow struct {
	Values []string
}

type TableAlign int

const (
	TableAlignLeft   TableAlign = 0
	TableAlignRight  TableAlign = 1
	TableAlignCenter TableAlign = 2
)

type TableHeader struct {
	Title string
	Align TableAlign
}

type Table struct {
	Headers []TableHeader
	Rows    []TableRow
}

func CreateTable(headers ...TableHeader) Table {
	return Table{
		Headers: headers,
		Rows:    []TableRow{},
	}
}

func (tbl *Table) AddRow(args ...string) {
	if len(tbl.Headers) != len(args) {
		fmt.Errorf("Header count didn't match row value count, ignoring row")
		return
	}

	row := TableRow{
		Values: args,
	}

	tbl.Rows = append(tbl.Rows, row)
}

func padding(amount int, spacer string, val int) string {
	if val < amount {
		return strings.Repeat(spacer, amount-val)
	}

	return ""
}

func align(alignment TableAlign, width int, str string) string {
	switch alignment {
	case TableAlignLeft:
		return str + padding(width, " ", len(str))
	case TableAlignCenter:
		p := padding(width/2, " ", len(str)/2)
		return p + str + p
	case TableAlignRight:
		return padding(width, " ", len(str)) + str
	}

	return str
}

func (tbl *Table) Render() string {
	widths := []int{}

	for i, _ := range tbl.Headers {
		rowWidth := len(tbl.Headers[i].Title)
		for _, row := range tbl.Rows {
			value := row.Values[i]
			if len(value) > rowWidth {
				rowWidth = len(value)
			}
		}

		widths = append(widths, rowWidth)
	}

	// Emit the header.
	result := ""
	for i, h := range tbl.Headers {
		if i > 0 {
			result += " | "
		}

		result += align(h.Align, widths[i], h.Title)
	}

	result += "\n"

	// Emit the line
	for i, _ := range tbl.Headers {
		if i > 0 {
			result += "-+-"
		}

		result += padding(widths[i], "-", 0)
	}

	// Emit the values
	for _, row := range tbl.Rows {
		result += "\n"

		for i, v := range row.Values {
			if i > 0 {
				result += " | "
			}

			result += align(tbl.Headers[i].Align, widths[i], v)
		}
	}

	return result
}
