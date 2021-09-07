package main

import (
	goxmppclient "xmpp-core/example/clients/go-xmpp-client"
	"xmpp-core/example/server"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xmppcore-test",
	Short: "xmppcore-test",
	Long:  "xmppcore-test",
	Run: func(cmd *cobra.Command, args []string) {
		s := server.New(&server.DefaultConfig)
		s.Start()
	},
}

var serverCmd = &cobra.Command{
	Use:   "start-server",
	Short: "start-server",
	Long:  "start-server",
	Run: func(cmd *cobra.Command, args []string) {
		s := server.New(&server.DefaultConfig)
		s.Start()
	},
}

var clientCmd = &cobra.Command{
	Use:   "start-client",
	Short: "start-client",
	Long:  "start-client",
	Run: func(cmd *cobra.Command, args []string) {
		goxmppclient.Start()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)
}

func main() {
	rootCmd.Execute()
}
