package xmppcore

import (
	"errors"
	"hash"

	"github.com/google/uuid"
	scramauth "github.com/yang-zzhong/scram-auth"
)

type MemoryAuthUser struct {
	id             string
	username       string
	salt           string
	iterationCount int

	passwords map[string]string
}

type MemoryPlainAuthUser struct {
	username string
	password string
}

func NewMemoryPlainAuthUser(username, password string) *MemoryPlainAuthUser {
	return &MemoryPlainAuthUser{username, password}
}

func (mpu *MemoryPlainAuthUser) Username() string {
	return mpu.username
}

func (mpu *MemoryPlainAuthUser) Password() string {
	return mpu.password
}

type MemoryPlainAuthUserFetcher struct {
	users []*MemoryPlainAuthUser
}

func NewMemoryPlainAuthUserFetcher() *MemoryPlainAuthUserFetcher {
	return &MemoryPlainAuthUserFetcher{[]*MemoryPlainAuthUser{}}
}

func (mpuf *MemoryPlainAuthUserFetcher) Add(u *MemoryPlainAuthUser) {
	mpuf.users = append(mpuf.users, u)
}

func (mpuf *MemoryPlainAuthUserFetcher) UserByUsername(username string) (PlainAuthUser, error) {
	for _, u := range mpuf.users {
		if u.Username() == username {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func NewMemomryAuthUser(username, password string, hash map[string]func() hash.Hash, ic int) *MemoryAuthUser {
	u := new(MemoryAuthUser)
	u.id = uuid.New().String()
	u.username = username
	u.iterationCount = ic
	u.passwords = make(map[string]string)
	u.salt = uuid.New().String()
	for name, hashBuilder := range hash {
		s := scramauth.NewServerScramAuth(hashBuilder, scramauth.None, nil)
		u.passwords[name] = string(s.SaltedPassword([]byte(password), []byte(u.salt), u.iterationCount))
	}
	return u
}

func (au *MemoryAuthUser) ID() string {
	return au.id
}

func (au *MemoryAuthUser) Username() string {
	return au.username
}

func (au *MemoryAuthUser) Salt() string {
	return au.salt
}

func (au *MemoryAuthUser) IterationCount() int {
	return au.iterationCount
}

func (au *MemoryAuthUser) Password(hashName string) (string, error) {
	p, ok := au.passwords[hashName]
	if !ok {
		return "", errors.New(ErrHashNotSupported)
	}
	return p, nil
}

type MemoryAuthUserFetcher struct {
	users []*MemoryAuthUser
}

func NewMemoryAuthUserFetcher() *MemoryAuthUserFetcher {
	return &MemoryAuthUserFetcher{users: []*MemoryAuthUser{}}
}

func (uf *MemoryAuthUserFetcher) Add(user *MemoryAuthUser) {
	uf.users = append(uf.users, user)
}
func (uf *MemoryAuthUserFetcher) UserByUsername(username string) (ScramAuthUser, error) {
	for _, u := range uf.users {
		if u.Username() == username {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

type MemoryAuthorized struct {
	parts []Part
}

func NewMemoryAuthorized() *MemoryAuthorized {
	return &MemoryAuthorized{parts: []Part{}}
}

func (ma *MemoryAuthorized) Authorized(_ string, part Part) {
	ma.parts = append(ma.parts, part)
}

func (ma *MemoryAuthorized) BindResource(part Part, resource string) (string, error) {
	part.Attr().JID.Resource = resource
	return part.Attr().JID.String(), nil
}

func (ma *MemoryAuthorized) FindPart(jid *JID) Part {
	for _, part := range ma.parts {
		if jid.Equal(&part.Attr().JID) {
			return part
		}
	}
	return nil
}
