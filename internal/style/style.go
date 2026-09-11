package style

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

var (
	CommandStyle = lipgloss.NewStyle().PaddingLeft(1)
	BoldStyle    = lipgloss.NewStyle().Bold(true)
	GovukBlue    = lipgloss.Color("#1d70b8")
)

// returns a lipgloss table designed for displaying key/value pairs
func KVTable() *table.Table {
	return tableWithBorders().StyleFunc(kvTableStyleFunc)
}

// returns a lipgloss table designed for displaying standard tabular data with a heading row
func ListTable(headers []string) *table.Table {
	return tableWithBorders().
		StyleFunc(listTableStyleFunc).
		Headers(headers...)
}

var tableBaseStyle = lipgloss.NewStyle().
	Padding(0, 1)

var kvTableKeyStyle = tableBaseStyle.
	Bold(true).
	Align(lipgloss.Left)

var listTableHeadingStyle = tableBaseStyle.
	Bold(true).
	Align(lipgloss.Center)

var listTableCellStyle = tableBaseStyle.
	Align(lipgloss.Left)

func kvTableStyleFunc(row int, col int) lipgloss.Style {
	switch col {
	case 0:
		return kvTableKeyStyle
	default:
		return tableBaseStyle
	}
}

func listTableStyleFunc(row int, col int) lipgloss.Style {
	switch row {
	case table.HeaderRow:
		return listTableHeadingStyle
	default:
		return listTableCellStyle
	}
}

func RenderHyperLink(url string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Underline(true).
		Hyperlink(url).Render(url)
}

func tableWithBorders() *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(GovukBlue))
}
