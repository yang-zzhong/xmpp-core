package client

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"

	xmppcore "github.com/yang-zzhong/xmpp-core"
)

type clientResourceBinder struct {
}

func (crb *clientResourceBinder) BindResource(part xmppcore.Part, resource string) (string, error) {
	part.Attr().JID.Resource = resource
	return part.Attr().JID.String(), nil
}

func Start() {
	conn, err := net.Dial("tcp", "localhost:5222")
	if err != nil {
		return
	}
	attr := xmppcore.PartAttr{
		JID:     xmppcore.JID{Domain: "hello-world.im", Resource: "/hello-world", Username: "test"},
		Version: "1.0",
		Domain:  "hello-world.im",
	}
	toAuth := xmppcore.NewScramToAuth("test", "123456", xmppcore.SM_SCRAM_SHA_256_PLUS, true)
	sasl := xmppcore.NewClientSASLFeature()
	sasl.Support(xmppcore.SM_SCRAM_SHA_256_PLUS, toAuth)
	logger := xmppcore.NewLogger(os.Stdout)
	client := xmppcore.NewClientPart(xmppcore.NewTcpConn(conn, true), logger, &attr)
	client.WithFeature(xmppcore.NewClientTlsFeature(&tls.Config{InsecureSkipVerify: true}))
	client.WithFeature(sasl)
	client.WithFeature(xmppcore.NewClientBindFeature(&clientResourceBinder{}, "xmpp-core-test"))
	// comp := xmppcore.NewClientCompressFeature()
	// comp.Support(xmppcore.ZLIB, func(rw io.ReadWriter) xmppcore.Compressor {
	// 	return xmppcore.NewCompZlib(rw)
	// })
	// client.WithFeature(comp)
	client.Channel().SetLogger(logger)
	if err := client.Negotiate(); err != nil {
		fmt.Printf("client negotiate error: %s\n", err.Error())
	}

	if err := <-client.Run(); err != nil {
		fmt.Printf("client error: %s\n", err.Error())
	}
	client.Stop()
}
