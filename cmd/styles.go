package cmd

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// Theme is the set of styles used for output. It's the place to hook in
// colour scheme support later.
type Theme struct {
	Heading     lipgloss.Style
	TableHeader lipgloss.Style
	IssueKey    lipgloss.Style
	// IssueTypes styles issue type names, keyed by lower-cased name.
	IssueTypes   map[string]lipgloss.Style
	PriorityHigh lipgloss.Style
	PriorityLow  lipgloss.Style
}

// newTheme builds the theme for a terminal with the given colour profile.
// Basic colours use the ANSI palette so they follow the terminal's own colour
// scheme. Subtle colours are given explicitly for each profile, as
// automatically downsampling muted true colours gives muddy results.
func newTheme(p colorprofile.Profile) Theme {
	c := lipgloss.Complete(p)
	fg := func(ansi, ansi256 int, truecolor string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(c(lipgloss.ANSIColor(ansi), lipgloss.ANSIColor(ansi256), lipgloss.Color(truecolor)))
	}
	return Theme{
		Heading:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow),
		TableHeader: lipgloss.NewStyle().Faint(true),
		IssueKey:    lipgloss.NewStyle().Foreground(lipgloss.Cyan),
		IssueTypes: map[string]lipgloss.Style{
			"story":      fg(2, 108, "#a3be8c"),
			"task":       fg(4, 110, "#81a1c1"),
			"sub-task":   fg(6, 109, "#8fbcbb"),
			"bug":        fg(1, 131, "#bf616a"),
			"incident":   fg(1, 131, "#bf616a"),
			"epic":       fg(5, 139, "#b48ead"),
			"initiative": fg(3, 173, "#d08770"),
		},
		PriorityHigh: fg(1, 131, "#bf616a"),
		PriorityLow:  fg(2, 108, "#a3be8c"),
	}
}

// IssueType renders an issue type name, uncoloured if the type is unknown.
func (t Theme) IssueType(name string) string {
	if s, ok := t.IssueTypes[strings.ToLower(name)]; ok {
		return s.Render(name)
	}
	return name
}

// Priority renders a priority name. Instances have many priority schemes
// (e.g. "High", "2-High", "1 - Highest"), so high and low are matched by
// substring; anything else is left uncoloured.
func (t Theme) Priority(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "high"):
		return t.PriorityHigh.Render(name)
	case strings.Contains(lower, "low"):
		return t.PriorityLow.Render(name)
	default:
		return name
	}
}
