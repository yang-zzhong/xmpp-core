package main

import (
	"xmpp-core/example"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "whoim",
	Short: "whoim",
	Long:  "whoim",
	Run: func(cmd *cobra.Command, args []string) {
		s := example.NewServer(&example.DefaultConfig)
		// time.AfterFunc(time.Duration(time.Second*5), func() {
		// 	s.Stop()
		// })
		s.Start()
	},
}

func main() {
	rootCmd.Execute()
}
