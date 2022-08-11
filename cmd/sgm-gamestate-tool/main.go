package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var headerFmt = color.New(color.FgGreen, color.Underline).SprintfFunc()
var columnFmt = color.New(color.FgYellow).SprintfFunc()

var rootCmd = &cobra.Command{
	Use:   "sgm-gamestate-tool",
	Short: "a tool to work with save games",
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	rootCmd.Execute()
}
