package xmppcore

import (
	"bytes"
	"fmt"
	"hash"
	"strings"

	"github.com/jackal-xmpp/stravaganza/v2"
	scramauth "github.com/yang-zzhong/scram-auth"
)

// rfc5802

const (
	ErrHashNotSupported = "hash not supported"
)

type ScramAuthUser interface {
	Salt() string
	Password(hashName string) (string, error)
	ID() string
	IterationCount() int
}

type ScramAuthUserFetcher interface {
	UserByUsername(username string) (ScramAuthUser, error)
}

type ScramAuth struct {
	hashBuild   func() hash.Hash
	useCB       bool
	userFetcher ScramAuthUserFetcher
	user        ScramAuthUser
}

func NewScramAuth(userFetcher ScramAuthUserFetcher, hashBuild func() hash.Hash, useCB bool) *ScramAuth {
	return &ScramAuth{
		hashBuild:   hashBuild,
		useCB:       useCB,
		userFetcher: userFetcher}
}

func (scram *ScramAuth) Auth(mechanism, authInfo string, part Part) (username string, err error) {
	var auth *scramauth.ServerScramAuth
	if scram.useCB {
		var buf bytes.Buffer
		if err := part.Conn().BindTlsUnique(&buf); err != nil {
			return "", err
		}
		fmt.Printf("server challenge bind string: %s\n", buf.String())
		auth = scramauth.NewServerScramAuth(scram.hashBuild, scramauth.TlsUnique, buf.Bytes())
	} else {
		auth = scramauth.NewServerScramAuth(scram.hashBuild, scramauth.None, []byte{})
	}
	r := bytes.NewBuffer([]byte(authInfo))
	var buf bytes.Buffer
	if err := auth.WriteChallengeMsg(r, func(username []byte) ([]byte, int, error) {
		if err := scram.initUser(username); err != nil {
			return nil, 0, err
		}
		return []byte(scram.user.Salt()), scram.user.IterationCount(), nil
	}, &buf); err != nil {
		return "", err
	}
	msg := stravaganza.NewBuilder("challenge").
		WithAttribute(stravaganza.Namespace, nsSASL).
		WithText(buf.String()).
		Build()
	if err = part.Channel().SendElement(msg); err != nil {
		return
	}
	if err = scram.waitChallengeResponse(&msg, part.Channel()); err != nil {
		return
	}
	hashName := scram.hashNameFromMechanism(mechanism)
	if err = scram.verifyPassword(auth, part, &msg, hashName); err != nil {
		return
	}
	return
}

func (scram *ScramAuth) initUser(username []byte) error {
	var err error
	scram.user, err = scram.userFetcher.UserByUsername(string(username))
	if err != nil {
		return SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	return nil
}

func (scram *ScramAuth) waitChallengeResponse(msg *stravaganza.Element, receiver Receiver) error {
	if err := receiver.NextElement(msg); err != nil {
		return err
	}
	if (*msg).Name() != "response" {
		return SaslFailureError(SFTemporaryAuthFailure, "not a response required")
	}
	return nil
}

func (scram *ScramAuth) hashNameFromMechanism(mechanism string) string {
	hashName := strings.Replace(mechanism, "SCRAM-", "", 1)
	return strings.Replace(hashName, "-PLUS", "", 1)
}

func (scram *ScramAuth) verifyPassword(auth *scramauth.ServerScramAuth, part Part, msg *stravaganza.Element, hashName string) error {
	password, err := scram.user.Password(hashName)
	if err != nil {
		return SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	r := bytes.NewBuffer([]byte((*msg).Text()))
	if err := auth.Verify(r, []byte(password)); err != nil {
		return SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	var buf bytes.Buffer
	if err := auth.WriteSignatureMsg(r, []byte(password), &buf); err != nil {
		return SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	return part.Channel().SendElement(stravaganza.NewBuilder("success").
		WithAttribute(stravaganza.Namespace, nsSASL).
		WithText(buf.String()).
		Build())
}
