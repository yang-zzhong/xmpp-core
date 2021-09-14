package xmppcore

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
	scramauth "github.com/yang-zzhong/scram-auth"
)

type ScramToAuth struct {
	authzid   string
	username  string
	password  string
	mechanism string
	useCB     bool
}

func NewScramToAuth(u, p string, mechanism string, useCB bool) *ScramToAuth {
	return &ScramToAuth{
		useCB:     useCB,
		authzid:   uuid.New().String(),
		mechanism: mechanism,
		username:  u, password: p}
}

func (sta *ScramToAuth) hashBuild() func() hash.Hash {
	switch sta.mechanism {
	case SM_SCRAM_SHA_1:
		return sha1.New
	case SM_SCRAM_SHA_1_PLUS:
		return sha1.New
	case SM_SCRAM_SHA_256:
		return sha256.New
	case SM_SCRAM_SHA_256_PLUS:
		return sha256.New
	case SM_SCRAM_SHA_512:
		return sha512.New
	case SM_SCRAM_SHA_512_PLUS:
		return sha512.New
	}
	return nil
}

func (sta *ScramToAuth) ToAuth(mechemism string, part Part) error {
	var auth *scramauth.ClientScramAuth
	hashBuild := sta.hashBuild()
	if hashBuild == nil {
		return errors.New("hash not supported")
	}
	if sta.useCB {
		var buf bytes.Buffer
		if err := part.Conn().BindTlsUnique(&buf); err != nil {
			return err
		}
		fmt.Printf("client challenge bind string: %s\n", buf.String())
		auth = scramauth.NewClientScramAuth(hashBuild, scramauth.TlsUnique, buf.Bytes())
	} else {
		auth = scramauth.NewClientScramAuth(hashBuild, scramauth.None, []byte{})
	}
	if err := sta.sendRequest(auth, part); err != nil {
		return err
	}
	cha, err := sta.challenge(part)
	if err != nil {
		return err
	}
	if err := sta.sendResponse(auth, cha, part); err != nil {
		return err
	}
	sign, err := sta.signature(part)
	if err != nil {
		return err
	}
	return auth.Verify(bytes.NewBuffer(sign), sta.password)
}

func (sta *ScramToAuth) signature(part Part) ([]byte, error) {
	var elem stravaganza.Element
	if err := part.Channel().NextElement(&elem); err != nil {
		return nil, err
	}
	if elem.Name() != "success" {
		return nil, errors.New("server failed auth")
	}
	return []byte(elem.Text()), nil
}

func (sta *ScramToAuth) sendResponse(auth *scramauth.ClientScramAuth, r io.Reader, part Part) error {
	var wr bytes.Buffer
	if err := auth.WriteResMsg(r, sta.password, &wr); err != nil {
		return err
	}
	elem := stravaganza.NewBuilder("response").
		WithAttribute("xmlns", nsSASL).
		WithText(wr.String()).Build()

	return part.Channel().SendElement(elem)
}

func (sta *ScramToAuth) challenge(part Part) (io.Reader, error) {
	var elem stravaganza.Element
	if err := part.Channel().NextElement(&elem); err != nil {
		return nil, err
	}
	if elem.Name() != "challenge" {
		return nil, errors.New("not a challenge element")
	}
	return bytes.NewBuffer([]byte(elem.Text())), nil
}

func (sta *ScramToAuth) sendRequest(auth *scramauth.ClientScramAuth, part Part) error {
	var buf bytes.Buffer
	if err := auth.WriteReqMsg(sta.authzid, sta.username, &buf); err != nil {
		return err
	}
	elem := stravaganza.NewBuilder("auth").
		WithAttribute("mechanism", sta.mechanism).
		WithAttribute("xmlns", nsSASL).WithText(buf.String()).Build()
	return part.Channel().SendElement(elem)
}
