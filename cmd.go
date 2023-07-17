package main

import (
	"fmt"

	"github.com/metarex-media/mrx-tool/decode"
	"github.com/metarex-media/mrx-tool/folderscan"
	"github.com/metarex-media/mrx-tool/versionstr"
	"github.com/spf13/cobra"
)

var UseLinkerOverrides string

func main() {

	doOverride := len(UseLinkerOverrides) > 1
	versionstr.Set(doOverride)

	rootCmd.SetUsageTemplate("empty" + rootCmd.UsageTemplate())

	// rootCmd.DebugFlags()
	cobra.CheckErr(rootCmd.Execute())
}

var rootCmd = &cobra.Command{
	Use:   "mrxtool",
	Short: "mrxtool - a simple CLI to manipulate mrx data",
	Long: `
Mrx Tool is a one stop command line tool for decoding and encoding mrx files.

Mrx Tool can:
- Genereate a yaml/json file giving a breakdown of the mrx file sructure and its contents. Using the "decode" key
- Extract mrx data and save it into files. using the "decodesave" key
- Encode mrx metadata into mrx files, given the files are in the same layout given by decode save. Using the "encode" key
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Long)
	},
}

// add the cpbra commands
func init() {
	// disable the unneeded completion opitions
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	//add the root commands
	rootCmd.AddCommand(decode.DecodeCmd)
	rootCmd.AddCommand(decode.DecodeSaveCmd)
	rootCmd.AddCommand(versionstr.VersionCmd)
	rootCmd.AddCommand(folderscan.EncodeCmd)
}
