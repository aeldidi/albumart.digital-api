package auth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

type jwtClaims struct {
	Issuer string `json:"iss"`
	// The UNIX timestamp when the token was issued.
	IssuedAt int64 `json:"iat"`
	// The UNIX timestamp when the token will expire.
	ExpirationTime int64 `json:"exp"`
}

type Transport struct {
	PrivateKey     *ecdsa.PrivateKey
	KeyId          string
	TeamID         string
	cached         string
	expirationTime time.Time
}

func (t *Transport) Token() string {
	if time.Now().Before(t.expirationTime) {
		return t.cached
	}

	log.Print("token expired: making a new one...")

	head := jwtHeader{
		Algorithm: "ES256",
		Type:      "JWT",
		KeyID:     t.KeyId,
	}

	now := time.Now()
	// A new token is generated each day
	t.expirationTime = now.AddDate(0, 0, 1)
	claims := jwtClaims{
		Issuer:         t.TeamID,
		IssuedAt:       now.Unix(),
		ExpirationTime: t.expirationTime.Unix(),
	}

	headJSON, err := json.Marshal(head)
	if err != nil {
		log.Fatalf("couldn't produce JWT header: %v", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		log.Fatalf("couldn't produce JWT claims: %v", err)
	}

	b64 := base64.RawURLEncoding
	msg := b64.EncodeToString(headJSON) + "." + b64.EncodeToString(claimsJSON)

	hasher := sha256.New()
	hasher.Write([]byte(msg))
	hash := sha256.Sum256([]byte(msg))

	r, s, err := ecdsa.Sign(rand.Reader, t.PrivateKey, hash[:])
	if err != nil {
		log.Fatalf("failed to sign JWT: %v", err)
	}

	sig := append(r.Bytes(), s.Bytes()...)
	t.cached = msg + "." + b64.EncodeToString(sig)

	log.Printf("new token expires: %v", t.expirationTime)
	return t.cached
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.Token())
	return http.DefaultTransport.RoundTrip(req)
}

func LoadPrivateKey(r io.Reader) (*ecdsa.PrivateKey, error) {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("couldn't decode PEM-encoded private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key.(*ecdsa.PrivateKey), nil
}
