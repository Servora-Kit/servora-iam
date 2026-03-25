package jwks

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	jwtpkg "github.com/Servora-Kit/servora/pkg/jwt"
)

const defaultCacheTTL = 5 * time.Minute

type Client struct {
	jwksURL    string
	mu         sync.RWMutex
	verifier   *jwtpkg.Verifier
	lastFetch  time.Time
	cacheTTL   time.Duration
	httpClient *http.Client
}

type ClientOption func(*Client)

func WithCacheTTL(d time.Duration) ClientOption {
	return func(c *Client) {
		c.cacheTTL = d
	}
}

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func NewClient(jwksURL string, opts ...ClientOption) *Client {
	c := &Client{
		jwksURL:    jwksURL,
		cacheTTL:   defaultCacheTTL,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Verifier() (*jwtpkg.Verifier, error) {
	c.mu.RLock()
	if c.verifier != nil && time.Since(c.lastFetch) < c.cacheTTL {
		v := c.verifier
		c.mu.RUnlock()
		return v, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.verifier != nil && time.Since(c.lastFetch) < c.cacheTTL {
		return c.verifier, nil
	}

	v, err := c.fetchAndBuild()
	if err != nil {
		return nil, err
	}
	c.verifier = v
	c.lastFetch = time.Now()
	return v, nil
}

func (c *Client) fetchAndBuild() (*jwtpkg.Verifier, error) {
	resp, err := c.httpClient.Get(c.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("jwks: fetch keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks: unexpected status %d from %s", resp.StatusCode, c.jwksURL)
	}

	var jwks Response
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("jwks: decode response: %w", err)
	}

	v := jwtpkg.NewVerifier()
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || key.Alg != "RS256" {
			continue
		}
		pub, err := jwkToRSAPublicKey(key)
		if err != nil {
			return nil, fmt.Errorf("jwks: parse key %s: %w", key.Kid, err)
		}
		v.AddKey(key.Kid, pub)
	}
	return v, nil
}

func jwkToRSAPublicKey(key Key) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

type discoveryResponse struct {
	JWKSURI string `json:"jwks_uri"`
}

func NewClientFromDiscovery(issuerURL string, opts ...ClientOption) (*Client, error) {
	hc := http.DefaultClient
	for _, opt := range opts {
		tmp := &Client{}
		opt(tmp)
		if tmp.httpClient != nil && tmp.httpClient != http.DefaultClient {
			hc = tmp.httpClient
		}
	}

	discURL := issuerURL + "/.well-known/openid-configuration"
	resp, err := hc.Get(discURL)
	if err != nil {
		return nil, fmt.Errorf("jwks: fetch discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks: discovery returned status %d", resp.StatusCode)
	}

	var disc discoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, fmt.Errorf("jwks: decode discovery: %w", err)
	}

	if disc.JWKSURI == "" {
		return nil, fmt.Errorf("jwks: discovery response missing jwks_uri")
	}

	return NewClient(disc.JWKSURI, opts...), nil
}
