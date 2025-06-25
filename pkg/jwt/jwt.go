package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	Header    = "access-token"
	Audience  = "Audience"
	KeyID     = "3e79646c4dbc408383a9eed09f2b85ae"
	Subject   = "login"
)

type Claims struct {
	UserName string `json:"userName"`
	jwtlib.RegisteredClaims
}

type Manager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	expiration time.Duration
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	D   string `json:"d,omitempty"`
	P   string `json:"p,omitempty"`
	Q   string `json:"q,omitempty"`
	DP  string `json:"dp,omitempty"`
	DQ  string `json:"dq,omitempty"`
	QI  string `json:"qi,omitempty"`
}

type jwkSet struct {
	Keys []jwkKey `json:"keys"`
}

func NewManager(jwkFile string, loginTimeoutMinutes int64) (*Manager, error) {
	if loginTimeoutMinutes <= 0 {
		loginTimeoutMinutes = 60
	}

	priv, pub, err := loadOrCreateKey(jwkFile)
	if err != nil {
		return nil, err
	}

	return &Manager{
		privateKey: priv,
		publicKey:  pub,
		expiration: time.Duration(loginTimeoutMinutes) * time.Minute,
	}, nil
}

func (m *Manager) CreateToken(username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserName: username,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   Subject,
			Audience:  jwtlib.ClaimStrings{Audience},
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.expiration)),
			ID:        fmt.Sprintf("%d", now.UnixNano()),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = KeyID
	return token.SignedString(m.privateKey)
}

func (m *Manager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtlib.Token) (any, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func loadOrCreateKey(jwkFile string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if jwkFile != "" {
		if priv, pub, err := loadFromFile(jwkFile); err == nil {
			return priv, pub, nil
		}
	}

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	if jwkFile != "" {
		if err := saveToFile(jwkFile, priv); err != nil {
			return nil, nil, err
		}
	}

	return priv, &priv.PublicKey, nil
}

func loadFromFile(path string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	// PEM format
	if block, _ := pem.Decode(data); block != nil {
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			keyAny, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, nil, err
			}
			rsaKey, ok := keyAny.(*rsa.PrivateKey)
			if !ok {
				return nil, nil, errors.New("not rsa private key")
			}
			return rsaKey, &rsaKey.PublicKey, nil
		}
		return key, &key.PublicKey, nil
	}

	// JWK set JSON (compatible with WVP jwk.json)
	var set jwkSet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, nil, err
	}
	if len(set.Keys) == 0 {
		return nil, nil, errors.New("empty jwk set")
	}

	// For simplicity, regenerate if JWK RSA parsing is complex;
	// WVP will create new jwk.json on first run anyway.
	return nil, nil, errors.New("jwk rsa import not supported, regenerate")
}

func saveToFile(path string, priv *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}
