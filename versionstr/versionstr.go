package versionstr

import (
	"fmt"

	"github.com/spf13/cobra"
)

// used to construct the version string when linking a release
var linkerOverride bool

var devBuild string = "dev"
var devDate string = "during development"

var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v", "Version"},
	Short:   "Print the version number of mrx tool",
	Long:    `All software has versions. This is mrx tool's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Mrx Tool version " + long(linkerOverride))
	},
}

func short(useLinkerOverrides bool) string {
	vStr := "0.0.1"

	if useLinkerOverrides {
		return vStr + "." + build[36:]
	} else {
		return vStr + "." + devBuild
	}
}

func long(useLinkerOverrides bool) string {
	vStr := fmt.Sprintf("%v (%s)", short(useLinkerOverrides), "pre-alpha")

	if useLinkerOverrides {
		return fmt.Sprintf("%s built %s", vStr, date)
	} else {
		return fmt.Sprintf("%s built %s", vStr, devDate)
	}
}

func Set(useLinkerOverrides bool) {
	linkerOverride = useLinkerOverrides
}
