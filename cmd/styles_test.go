package cmd

import (
	"testing"

	"github.com/charmbracelet/colorprofile"
)

func TestThemePriority(t *testing.T) {
	theme := newTheme(colorprofile.TrueColor)
	cases := map[string]string{
		"High":        theme.PriorityHigh.Render("High"),
		"1 - Highest": theme.PriorityHigh.Render("1 - Highest"),
		"4-Low":       theme.PriorityLow.Render("4-Low"),
		"Lowest":      theme.PriorityLow.Render("Lowest"),
		"Medium":      "Medium",
		"Blocker":     "Blocker",
	}
	for name, want := range cases {
		if got := theme.Priority(name); got != want {
			t.Errorf("Priority(%q) = %q, want %q", name, got, want)
		}
	}
}
