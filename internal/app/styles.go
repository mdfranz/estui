package app

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

type appStyles struct {
	statusBar          lipgloss.Style
	errorText          lipgloss.Style
	editorBox          lipgloss.Style
	optBar             lipgloss.Style
	optLabel           lipgloss.Style
	optValue           lipgloss.Style
	optActive          lipgloss.Style
	timeBar            lipgloss.Style
	timeActive         lipgloss.Style
	timeInactive       lipgloss.Style
	hintHeader         lipgloss.Style
	hintRow            lipgloss.Style
	hintRowSelected    lipgloss.Style
	hintSugHeader      lipgloss.Style
	hintSugRow         lipgloss.Style
	recordHeader       lipgloss.Style
	recordField        lipgloss.Style
	recordSep          lipgloss.Style
	recordCursorLine   lipgloss.Style
	recordSelectedMark lipgloss.Style
	tableStyle         table.Styles
}

func newStyles() *appStyles {
	base := table.DefaultStyles()
	base.Header = base.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	base.Selected = base.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	return &appStyles{
		statusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1),
		errorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		editorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
		recordHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 0, 0, 2),
		recordField: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		recordSep: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		optBar: lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("250")),
		optLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Bold(true),
		optValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("238")).
			Padding(0, 1),
		optActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1),
		timeBar: lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("250")).
			Padding(0, 1),
		timeActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1),
		timeInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Background(lipgloss.Color("238")).
			Padding(0, 1),
		recordCursorLine: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true),
		recordSelectedMark: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true),
		hintHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(1, 0, 0, 2),
		hintRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			PaddingLeft(2),
		hintRowSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			PaddingLeft(2),
		hintSugHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true),
		hintSugRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		tableStyle: base,
	}
}
