/*
xmpp_echo is a demo client that connect on an XMPP server and echo message received back to original sender.
*/

package goxmppclient

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackal-xmpp/stravaganza/v2"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
)

func Start() {
	config := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address:   "localhost:5222",
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Jid:          "test@localhost",
		Credential:   xmpp.Password("123456"),
		StreamLogger: os.Stdout,
		Insecure:     true,
	}

	router := xmpp.NewRouter()
	router.HandleFunc("message", handleMessage)

	client, err := xmpp.NewClient(&config, router, errorHandler)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	time.AfterFunc(3*time.Second, func() {
		msg := stravaganza.NewBuilder("message").
			WithAttribute("from", "test@localhost").
			WithAttribute("to", "romeo@localhost").
			WithAttribute("type", "chat").
			WithAttribute("xml:lang", "en").
			WithChild(stravaganza.NewBuilder("body").WithText("Art thou not Romeo, and a Montague?").Build()).Build()
		client.SendRaw(msg.GoString())
	})

	// If you pass the client to a connection manager, it will handle the reconnect policy
	// for you automatically.
	cm := xmpp.NewStreamManager(client, nil)
	log.Fatal(cm.Run())
}

func handleMessage(s xmpp.Sender, p stanza.Packet) {
	msg, ok := p.(stanza.Message)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", msg.Body, msg.From)
	reply := stanza.Message{Attrs: stanza.Attrs{To: msg.From}, Body: msg.Body}
	_ = s.Send(reply)
}

func errorHandler(err error) {
	fmt.Println(err.Error())
}
