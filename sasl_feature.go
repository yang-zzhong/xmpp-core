package xmppcore

import (
	"errors"
	"fmt"
	"strings"

	"encoding/base64"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type Auth interface {
	Auth(mechanism, authInfo string, part Part) (username string, err error)
}

type Authorized interface {
	Authorized(username string, part Part)
}

func AuthPayload(encoded string, payload *string) error {
	if len(encoded) == 0 {
		return SaslFailureError(SFIncorrectEncoding, "")
	}
	if c, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		*payload = string(c)
	} else {
		return SaslFailureError(SFIncorrectEncoding, "")
	}
	return nil
}

func SaslFailureElem(tagName, desc string) stravaganza.Element {
	err := []stravaganza.Element{stravaganza.NewBuilder(tagName).Build()}
	if desc != "" {
		err = append(err, stravaganza.NewBuilder("text").WithText(desc).WithAttribute("xml:lang", "en").Build())
	}
	return stravaganza.NewBuilder("failure").WithAttribute("xmlns", nsSASL).WithChildren(err...).Build()
}

func SaslFailureError(tagName, desc string) error {
	tagName = strings.ReplaceAll(tagName, "-", " ")
	if desc == "" {
		return errors.New(tagName)
	}
	return errors.New(fmt.Sprintf("%s: %s", tagName, desc))
}

var AllSaslFailures = []string{
	SFAborted,
	SFAccountDisabled,
	SFCredentialsExpired,
	SFEncryptionRequired,
	SFIncorrectEncoding,
	SFInvalidAuthzid,
	SFInvalidMechanism,
	SFMalformedRequest,
	SFMechanismTooWeak,
	SFNotAuthorized,
	SFTemporaryAuthFailure}

func SaslFailureElemFromError(err error) stravaganza.Element {
	ea := strings.Split(err.Error(), ":")
	f := strings.ReplaceAll(ea[0], " ", "-")
	desc := ""
	if len(ea) > 0 {
		desc = ea[1]
	}
	for _, failure := range AllSaslFailures {
		if failure == f {
			return SaslFailureElem(failure, desc)
		}
	}
	return SaslFailureElem(SFTemporaryAuthFailure, "")
}

const (
	nsSASL = "urn:ietf:params:xml:ns:xmpp-sasl"

	SM_EXTERNAL           = "EXTERNAL"           // where authentication is implicit in the context (e.g., for protocols already using IPsec or TLS)
	SM_ANONYMOUS          = "ANONYMOUS"          // for unauthenticated guest access
	SM_PLAIN              = "PLAIN"              // a simple cleartext password mechanism, defined in RFC 4616
	SM_OTP                = "OTP"                // a one-time password mechanism. Obsoletes the SKEY mechanism
	SM_SKEY               = "SKEY"               // an S/KEY mechanism
	SM_DIGEST_MD5         = "DIGEST-MD5"         // partially HTTP Digest compatible challenge-response scheme based upon MD5. DIGEST-MD5 offered a data security layer.
	SM_SCRAM_SHA_1        = "SCRAM-SHA-1"        // (RFC 5802), modern challenge-response scheme based mechanism with channel binding support
	SM_SCRAM_SHA_1_PLUS   = "SCRAM-SHA-1-PLUS"   // (RFC 5802), modern challenge-response scheme based mechanism with channel binding support
	SM_SCRAM_SHA_256      = "SCRAM-SHA-256"      // (RFC 5802), modern challenge-response scheme based mechanism with channel binding support
	SM_SCRAM_SHA_256_PLUS = "SCRAM-SHA-256-PLUS" // (RFC 5802), modern challenge-response scheme based mechanism with channel binding support
	SM_SCRAM_SHA_512      = "SCRAM-SHA-512"      // (RFC 5802), modern challenge-response scheme based mechanism with channel binding support
	SM_SCRAM_SHA_512_PLUS = "SCRAM-SHA-512-PLUS" // (RFC 5802), modern challenge-response scheme based mechanism with channel binding support
	SM_NTLM               = "NTLM"               // an NT LAN Manager authentication mechanism
	SM_GS2_               = "GS2-"               // family of mechanisms supports arbitrary GSS-API mechanisms in SASL.[3] It is now standardized as RFC 5801.
	SM_GSSAPI             = "GSSAPI"             // for Kerberos V5 authentication via the GSSAPI. GSSAPI offers a data-security layer.
	SM_BROWSERID_AES128   = "BROWSERID-AES128"   // for Mozilla Persona authentication
	SM_EAP_AES128         = "EAP-AES128"         // for GSS EAP authentication
	SM_OAUTH_1            = "OAUTH-1"            // bearer tokens (RFC 6750), communicated through TLS
	SM_OAUTH_2            = "OAUTH-2"            // bearer tokens (RFC 6750), communicated through TLS

	SFAborted              = "aborted"
	SFAccountDisabled      = "account-disabled"
	SFCredentialsExpired   = "credentials-expired"
	SFEncryptionRequired   = "encryption-required"
	SFIncorrectEncoding    = "incorrect-encoding"
	SFInvalidAuthzid       = "invalid-authzid"
	SFInvalidMechanism     = "invalid-mechanism"
	SFMalformedRequest     = "malformed-request"
	SFMechanismTooWeak     = "mechanism-too-weak"
	SFNotAuthorized        = "not-authorized"
	SFTemporaryAuthFailure = "temporary-auth-failure"
)

type SASLFeature struct {
	supported  map[string]Auth
	authorized Authorized
}

func NewSASLFeature(authorized Authorized) *SASLFeature {
	mf := new(SASLFeature)
	mf.supported = make(map[string]Auth)
	mf.authorized = authorized
	return mf
}

func (mf *SASLFeature) notifySupported(s GoingStream) error {
	if len(mf.supported) == 0 {
		return nil
	}
	msg := stravaganza.NewBuilder("features").
		WithChild(mf.Elem()).Build()

	return s.SendElement(msg)
}

func (mf *SASLFeature) Elem() stravaganza.Element {
	ms := []stravaganza.Element{}
	for name := range mf.supported {
		ms = append(ms, stravaganza.NewBuilder("mechanism").WithText(name).Build())
	}
	return stravaganza.NewBuilder("mechanisms").
		WithAttribute("xmlns", nsSASL).
		WithChildren(ms...).Build()
}

func (mf *SASLFeature) Support(name string, auth Auth) *SASLFeature {
	mf.supported[name] = auth
	return mf
}

func (mf *SASLFeature) Unsupport(name string) *SASLFeature {
	delete(mf.supported, name)
	return mf
}

func (mf *SASLFeature) Resolve(part Part) error {
	if err := mf.notifySupported(part.GoingStream()); err != nil {
		return err
	}
	as, mech, err := mf.clientRequest(part)
	if err != nil {
		return err
	}
	auth, ok := mf.supported[mech]
	if !ok {
		supported := []string{}
		for k := range mf.supported {
			supported = append(supported, k)
		}
		desc := fmt.Sprintf("only support [%s] in this server, client preffers [%s]", strings.Join(supported, ","), mech)
		part.GoingStream().SendElement(SaslFailureElem(SFInvalidMechanism, desc))
		return SaslFailureError(SFInvalidMechanism, desc)
	}
	username, err := auth.Auth(mech, as, part)
	if err != nil {
		part.GoingStream().SendElement(SaslFailureElemFromError(err))
		return err
	}
	part.Attr().JID.Username = username
	part.Attr().JID.Domain = part.Attr().Domain
	mf.authorized.Authorized(username, part)
	return nil
}

func (mf *SASLFeature) clientRequest(part Part) (as string, mechanism string, err error) {
	elem, e := mf.nextElement(part)
	if e != nil {
		err = e
		return
	}
	if elem.Name() != "auth" {
		err = SaslFailureError(SFIncorrectEncoding, "not a auth elem from client")
		part.GoingStream().SendElement(SaslFailureElemFromError(err))
		return
	}
	as = elem.Text()
	mechanism = elem.Attribute("mechanism")
	return
}

func (mf *SASLFeature) nextElement(part Part) (stravaganza.Element, error) {
	var elem stravaganza.Element
	if err := part.CommingStream().NextElement(&elem); err != nil {
		return elem, err
	}
	if elem.Name() == "abort" {
		part.GoingStream().SendElement(SaslFailureElem(SFAborted, ""))
		return nil, SaslFailureError(SFAborted, "")
	}
	return elem, nil
}
