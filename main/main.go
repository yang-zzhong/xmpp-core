package main

import (
	"github.com/yang-zzhong/xmpp-core/example/clients/client"
	goxmppclient "github.com/yang-zzhong/xmpp-core/example/clients/go-xmpp-client"
	"github.com/yang-zzhong/xmpp-core/example/server"

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

var xmppClientCmd = &cobra.Command{
	Use:   "start-xmpp-client",
	Short: "start-xmpp-client",
	Long:  "start-xmpp-client",
	Run: func(cmd *cobra.Command, args []string) {
		goxmppclient.Start()
	},
}

var clientCmd = &cobra.Command{
	Use:   "start-client",
	Short: "start-client",
	Long:  "start-client",
	Run: func(cmd *cobra.Command, args []string) {
		client.Start()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)
	rootCmd.AddCommand(xmppClientCmd)
}

func main() {
	rootCmd.Execute()
}
