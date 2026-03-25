package jwks

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"

	jwtpkg "github.com/Servora-Kit/servora/pkg/jwt"
)

type Response struct {
	Keys []Key `json:"keys"`
}

type Key struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type KeyManager struct {
	signer *jwtpkg.Signer
}

type options struct {
	privateKeyPath string
	privateKeyPEM  []byte
}

type Option func(*options)

func WithPrivateKeyPath(path string) Option {
	return func(o *options) {
		o.privateKeyPath = path
	}
}

func WithPrivateKeyPEM(pem []byte) Option {
	return func(o *options) {
		o.privateKeyPEM = pem
	}
}

func NewKeyManager(opts ...Option) (*KeyManager, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	var pemData []byte
	switch {
	case o.privateKeyPath != "":
		data, err := os.ReadFile(o.privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("jwks: read private key file: %w", err)
		}
		pemData = data
	case len(o.privateKeyPEM) > 0:
		pemData = o.privateKeyPEM
	default:
		return nil, fmt.Errorf("jwks: no private key provided")
	}

	signer, err := jwtpkg.NewSigner(pemData)
	if err != nil {
		return nil, fmt.Errorf("jwks: create signer: %w", err)
	}

	return &KeyManager{signer: signer}, nil
}

func (km *KeyManager) Signer() *jwtpkg.Signer {
	return km.signer
}

func (km *KeyManager) Verifier() *jwtpkg.Verifier {
	v := jwtpkg.NewVerifier()
	v.AddKey(km.signer.KID(), km.signer.PublicKey())
	return v
}

func (km *KeyManager) JWKSResponse() *Response {
	return &Response{
		Keys: []Key{rsaPublicKeyToJWK(km.signer.PublicKey(), km.signer.KID())},
	}
}

func rsaPublicKeyToJWK(pub *rsa.PublicKey, kid string) Key {
	return Key{
		Kty: "RSA",
		Use: "sig",
		Kid: kid,
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
