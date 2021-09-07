package xmppcore

type C2S struct {
	tls      *TlsFeature
	sasl     *SASLFeature
	bind     *BindFeature
	compress *CompressionFeature
	part     *XPart
}

func NewC2S(conn Conn, domain string, logger Logger) *C2S {
	return &C2S{part: NewXPart(conn, domain, logger)}
}

func (c2s *C2S) WithTLS(certFile, keyFile string) *C2S {
	c2s.tls = NewTlsFeature(certFile, keyFile)
	return c2s
}

func (c2s *C2S) WithSASLFeature(authed Authorized) *C2S {
	c2s.sasl = NewSASLFeature(authed)
	return c2s
}

func (c2s *C2S) WithSASLSupport(name string, auth Auth) *C2S {
	c2s.sasl.Support(name, auth)
	return c2s
}

func (c2s *C2S) WithBind() *C2S {
	c2s.bind = NewBindFeature()
	return c2s
}

func (c2s *C2S) WithCompressSupport(name string, build BuildCompressor) *C2S {
	if c2s.compress == nil {
		c2s.compress = NewCompressFeature()
	}
	c2s.compress.Support(name, build)
	return c2s
}

func (c2s *C2S) Part() Part {
	return c2s.part
}

func (c2s *C2S) WithElemHandler(handler ElemHandler) *C2S {
	c2s.part.WithElemHandler(handler)
	return c2s
}

func (c2s *C2S) HandleStandFeature() {
	if c2s.tls != nil {
		c2s.part.WithRequiredFeature(c2s.tls)
	}
	if c2s.sasl != nil {
		c2s.part.WithRequiredFeature(c2s.sasl)
	}
	if c2s.bind != nil {
		c2s.part.WithOptionalFeature(c2s.bind)
	}
	if c2s.compress != nil {
		c2s.part.WithOptionalFeature(c2s.compress)
	}
}

func (c2s *C2S) Start() error {
	c2s.HandleStandFeature()
	return c2s.part.Run()
}
