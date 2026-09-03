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
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(GovukBlue)).
		StyleFunc(kvTableStyleFunc)
	return t
}

var kvTableBaseStyle = lipgloss.NewStyle().
	Padding(0, 1)

var kvTableKeyStyle = kvTableBaseStyle.
	Bold(true).
	Align(lipgloss.Left)

func kvTableStyleFunc(row int, col int) lipgloss.Style {
	switch col {
	case 0:
		return kvTableKeyStyle
	default:
		return kvTableBaseStyle
	}
}

func RenderHyperLink(url string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Underline(true).
		Hyperlink(url).Render(url)
}
