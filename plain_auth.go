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
	var payload string
	if err = AuthPayload(authInfo, &payload); err != nil {
		return
	}
	res := bytes.Split([]byte(payload), []byte{0x00})
	if len(res) != 3 {
		err = SaslFailureError(SFIncorrectEncoding, "")
		return
	}
	username = string(res[1])
	password := string(res[2])
	user, err := auth.userFetcher.UserByUsername(username)
	if err != nil {
		return "", SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	h := auth.hashCreate()
	hp := h.Sum([]byte(password))
	if user.Password() != string(hp) {
		return "", SaslFailureError(SFTemporaryAuthFailure, "password error")
	}
	part.GoingStream().SendElement(stravaganza.NewBuilder("success").
		WithAttribute("xmlns", nsSASL).
		WithText(base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))).Build())
	return username, nil
}