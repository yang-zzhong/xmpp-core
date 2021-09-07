package xmppcore

type S2S struct {
	tls  *TlsFeature
	sasl *SASLFeature
	part *XPart
}

func NewS2S(conn Conn, domain string, logger Logger) *S2S {
	return &S2S{part: NewXPart(conn, domain, logger)}
}

func (s2s *S2S) WithTLS(certFile, keyFile string) *S2S {
	s2s.tls = NewTlsFeature(certFile, keyFile)
	return s2s
}

func (s2s *S2S) WithSASLFeature(authed Authorized) *S2S {
	s2s.sasl = NewSASLFeature(authed)
	return s2s
}

func (s2s *S2S) WithSASLSupport(name string, auth Auth) *S2S {
	s2s.sasl.Support(name, auth)
	return s2s
}

func (s2s *S2S) WithElemHandler(handler ElemHandler) *S2S {
	s2s.part.WithElemHandler(handler)
	return s2s
}

func (s2s *S2S) HandleStandFeature() {
	if s2s.tls != nil {
		s2s.part.WithRequiredFeature(s2s.tls)
	}
	if s2s.sasl != nil {
		s2s.part.WithRequiredFeature(s2s.sasl)
	}
}

func (s2s *S2S) Start() error {
	s2s.HandleStandFeature()
	return s2s.part.Run()
}
