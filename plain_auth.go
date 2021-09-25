package xmppcore

import (
	"bytes"
	"encoding/base64"
	"hash"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
)

type PlainAuth struct {
	userFetcher PlainAuthUserFetcher
	hashCreate  func() hash.Hash
}

type PlainAuthUser interface {
	Username() string
	Password() string
}

type PlainAuthUserFetcher interface {
	UserByUsername(string) (PlainAuthUser, error)
}

func NewPlainAuth(uf PlainAuthUserFetcher, hashCreate func() hash.Hash) *PlainAuth {
	return &PlainAuth{uf, hashCreate}
}

func (auth *PlainAuth) Auth(mechanism, authInfo string, part Part) (username string, err error) {
	var password string
	if err = auth.decodePayload(authInfo, &username, &password); err != nil {
		return
	}
	user, err := auth.userFetcher.UserByUsername(username)
	if err != nil {
		return "", SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	h := auth.hashCreate()
	hp := h.Sum([]byte(password))
	if user.Password() != string(hp) {
		return "", SaslFailureError(SFTemporaryAuthFailure, "password error")
	}
	part.Channel().SendElement(stravaganza.NewBuilder("success").
		WithAttribute("xmlns", NSSasl).
		WithText(base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))).Build())
	return username, nil
}

func (auth *PlainAuth) decodePayload(authInfo string, username, password *string) error {
	var payload string
	if err := AuthPayload(authInfo, &payload); err != nil {
		return err
	}
	res := bytes.Split([]byte(payload), []byte{0x00})
	if len(res) != 3 {
		return SaslFailureError(SFIncorrectEncoding, "")
	}
	*username = string(res[1])
	*password = string(res[2])
	return nil
}
