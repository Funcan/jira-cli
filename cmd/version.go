package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// The version details are read from the build info Go records in the binary
// when building from a git checkout: the module version comes from the tag on
// HEAD (a pseudo-version if HEAD is past it, with "+dirty" for uncommitted
// changes), and the commit and its time from git.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		Version = v
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			Commit = s.Value
		case "vcs.time":
			Date = s.Value
		}
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wiki-lint %s (%s from %s)\n", Version, Commit, Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
