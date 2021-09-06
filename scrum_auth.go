package xmppcore

import (
	"bytes"
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
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

type scramParam struct {
	username  string
	cNonce    string
	gs2Header string
	mechanism string
	params    map[string]string
}

func (p *scramParam) string() string {
	ps := []string{}
	for k, v := range p.params {
		ps = append(ps, k+"="+v)
	}
	return strings.Join(ps, ",")
}

type ScramAuth struct {
	channelBinding bool
	userFetcher    ScramAuthUserFetcher
	hashBuilder    func() hash.Hash

	param        *scramParam
	user         ScramAuthUser
	sNonce       string
	firstMessage string
}

func NewScramAuth(channelBinding bool, userFetcher ScramAuthUserFetcher, hashBuilder func() hash.Hash) *ScramAuth {
	return &ScramAuth{
		channelBinding: channelBinding,
		userFetcher:    userFetcher,
		hashBuilder:    hashBuilder}
}

func (scram *ScramAuth) Auth(mechanism, authInfo string, part Part) (username string, err error) {
	var payload string
	if err = AuthPayload(authInfo, &payload); err != nil {
		return
	}
	if err = scram.parseAuthInfo(authInfo); err != nil {
		return
	}
	if err = scram.initUser(); err != nil {
		return
	}
	var msg stravaganza.Element
	if err = scram.prepareChallengeMessage(&msg); err != nil {
		return
	}
	part.GoingStream().SendElement(msg)
	if err = scram.waitChallengeResponse(&msg, part.CommingStream()); err != nil {
		return
	}
	hashName := scram.hashNameFromMechanism(mechanism)
	if err = scram.verifyPassword(part, &msg, hashName); err != nil {
		return
	}
	return
}

func (scram *ScramAuth) initUser() error {
	p := scram.param
	if username, uok := p.params["n"]; uok {
		var err error
		scram.user, err = scram.userFetcher.UserByUsername(username)
		if err != nil {
			return SaslFailureError(SFTemporaryAuthFailure, err.Error())
		}
	}
	return SaslFailureError(SFMalformedRequest, "has not u in params")
}

func (scram *ScramAuth) prepareChallengeMessage(msg *stravaganza.Element) error {
	username, uok := scram.param.params["n"]
	cNonce, cok := scram.param.params["r"]
	if !uok || !cok || username == "" || cNonce == "" {
		return SaslFailureError(SFMalformedRequest, "")
	}
	saltBytes, err := base64.RawURLEncoding.DecodeString(scram.user.Salt())
	if err != nil {
		return SaslFailureError(SFTemporaryAuthFailure, "")
	}
	buf := bytes.NewBuffer(saltBytes)
	buf.WriteString(scram.user.ID())
	pepperedSaltB64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	scram.sNonce = cNonce + "-" + uuid.New().String()
	scram.firstMessage = fmt.Sprintf("r=%s,s=%s,i=%d", scram.sNonce, pepperedSaltB64, scram.user.IterationCount())

	*msg = stravaganza.NewBuilder("challenge").
		WithAttribute(stravaganza.Namespace, nsSASL).
		WithText(base64.StdEncoding.EncodeToString([]byte(scram.firstMessage))).
		Build()
	return nil
}

func (scram *ScramAuth) waitChallengeResponse(msg *stravaganza.Element, receiver CommingStream) error {
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

func (scram *ScramAuth) verifyPassword(part Part, msg *stravaganza.Element, hashName string) error {
	var payload string
	if err := AuthPayload((*msg).Text(), &payload); err != nil {
		return err
	}
	var c string
	if err := scram.bindInputStr(part, &c); err != nil {
		return err
	}
	v, err := scram.serverSignature(payload, c, hashName)
	if err != nil {
		return err
	}
	return part.GoingStream().SendElement(stravaganza.NewBuilder("success").
		WithAttribute(stravaganza.Namespace, nsSASL).
		WithText(base64.StdEncoding.EncodeToString([]byte(v))).
		Build())
}

func (scram *ScramAuth) serverSignature(payload, channelBinding, hashName string) (string, error) {
	clientFinalMessageBare := fmt.Sprintf("c=%s,r=%s", channelBinding, scram.sNonce)
	password, err := scram.user.Password(hashName)
	if err != nil {
		return "", SaslFailureError(SFTemporaryAuthFailure, err.Error())
	}
	clientKey := scram.hmac([]byte("Client Key"), []byte(password))
	storedKey := scram.hash(clientKey)
	initialMessage := scram.param.string()
	authMessage := initialMessage + "," + scram.firstMessage + "," + clientFinalMessageBare
	clientSignature := scram.hmac([]byte(authMessage), storedKey)

	clientProof := make([]byte, len(clientKey))
	for i := 0; i < len(clientKey); i++ {
		clientProof[i] = clientKey[i] ^ clientSignature[i]
	}
	serverKey := scram.hmac([]byte("Server Key"), []byte(password))
	serverSignature := scram.hmac([]byte(authMessage), []byte(serverKey))

	clientFinalMessage := clientFinalMessageBare + ",p=" + base64.StdEncoding.EncodeToString(clientProof)
	if clientFinalMessage != payload {
		return "", SaslFailureError(SFNotAuthorized, "")
	}
	return "v=" + base64.StdEncoding.EncodeToString(serverSignature), nil
}

func (scram *ScramAuth) bindInputStr(part Part, c *string) error {
	buf := new(bytes.Buffer)
	buf.Write([]byte(scram.param.gs2Header))
	if scram.channelBinding {
		switch scram.param.mechanism {
		case "tls-unique":
			if err := part.Conn().BindTlsUnique(buf); err != nil {
				return err
			}
		}
	}
	*c = base64.StdEncoding.EncodeToString(buf.Bytes())
	return nil
}

func (scram *ScramAuth) parseAuthInfo(authInfo string) error {
	b, err := base64.StdEncoding.DecodeString(authInfo)
	if err != nil {
		return SaslFailureError(SFIncorrectEncoding, "")
	}
	scram.param = &scramParam{params: make(map[string]string)}
	sp := strings.Split(string(b), ",")
	if len(sp) < 2 {
		return SaslFailureError(SFIncorrectEncoding, "")
	}
	gs2BindFlag := sp[0]
	switch gs2BindFlag {
	case "p":
		// Channel binding is supported and required.
		if !scram.channelBinding {
			return SaslFailureError(SFNotAuthorized, "")
		}
	case "n", "y":
		// Channel binding is not supported, or is supported but is not required.
	default:
		if !strings.HasPrefix(gs2BindFlag, "p=") {
			return SaslFailureError(SFMalformedRequest, "")
		}
		if !scram.channelBinding {
			return SaslFailureError(SFNotAuthorized, "")
		}
		scram.param.mechanism = gs2BindFlag[2:]
	}
	username := sp[1]
	scram.param.gs2Header = gs2BindFlag + "," + username + ","
	if len(username) > 0 {
		unn := strings.Split(username, "=")
		if unn[0] != "a" {
			return SaslFailureError(SFMalformedRequest, "")
		}
		scram.param.username = unn[1]
	}
	for i := 2; i < len(sp); i++ {
		unn := strings.Split(username, "=")
		scram.param.params[unn[0]] = unn[1]
	}
	return nil
}

func (scram *ScramAuth) hmac(b []byte, key []byte) []byte {
	m := hmac.New(scram.hashBuilder, key)
	m.Write(b)
	return m.Sum(nil)
}

func (scram *ScramAuth) hash(b []byte) []byte {
	h := scram.hashBuilder()
	h.Write(b)
	return h.Sum(nil)
}
