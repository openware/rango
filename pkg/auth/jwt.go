package auth

import (
	"crypto/rsa"
	"encoding/json"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
)

// Auth struct represents parsed jwt information.
type Auth struct {
	UID        string      `json:"uid"`
	State      string      `json:"state"`
	Email      string      `json:"email"`
	Role       string      `json:"role"`
	ReferralID json.Number `json:"referral_id"`
	Level      json.Number `json:"level"`
	Audience   []string    `json:"aud,omitempty"`

	jwt.StandardClaims
}

// ParseAndValidate parses token and validates it's jwt signature with given key.
func ParseAndValidate(token string, key *rsa.PublicKey) (Auth, error) {
	auth := Auth{}

	_, err := jwt.ParseWithClaims(token, &auth, func(t *jwt.Token) (interface{}, error) {
		return key, nil
	})

	return auth, err
}

func appendClaims(defaultClaims, customClaims jwt.MapClaims) jwt.MapClaims {
	if defaultClaims == nil {
		return customClaims
	}

	if customClaims == nil {
		return defaultClaims
	}

	for k, v := range customClaims {
		defaultClaims[k] = v
	}

	return defaultClaims
}

// ForgeToken creates a valid JWT signed by the given private key
func ForgeToken(uid, email, role string, level int, key *rsa.PrivateKey, customClaims jwt.MapClaims) (string, error) {
	claims := appendClaims(jwt.MapClaims{
		"iat":         time.Now().Unix(),
		"jti":         strconv.FormatInt(time.Now().Unix(), 10),
		"exp":         time.Now().UTC().Add(time.Hour).Unix(),
		"sub":         "session",
		"iss":         "barong",
		"aud":         [2]string{"peatio", "barong"},
		"uid":         uid,
		"email":       email,
		"role":        role,
		"level":       level,
		"state":       "active",
		"referral_id": nil,
	}, customClaims)

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	return t.SignedString(key)
}
