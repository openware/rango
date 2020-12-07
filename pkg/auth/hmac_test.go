package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIKeyHMACGetSignature(t *testing.T) {
	accessKey := "61d025b8573501c2"
	secretKey := "2d0b4979c7fe6986daa8e21d1dc0644f"

	k := NewAPIKeyHMAC(accessKey, secretKey)
	nonce := int64(1584524005143)
	signature := k.GetSignature(nonce)
	assert.Equal(t, "bd42b945e095880e28d046846dbecf655fdf09d95a396a24fe6fe1df42f15d13", signature)
}

func TestAPIKeyHMACGetSignedHeader(t *testing.T) {
	accessKey := "61d025b8573501c2"
	secretKey := "2d0b4979c7fe6986daa8e21d1dc0644f"

	k := NewAPIKeyHMAC(accessKey, secretKey)
	nonce := int64(1584524005143)
	headers := k.GetSignedHeader(nonce)

	assert.Equal(t,
		http.Header{
			"X-Auth-Apikey":    {accessKey},
			"X-Auth-Nonce":     {"1584524005143"},
			"X-Auth-Signature": {"bd42b945e095880e28d046846dbecf655fdf09d95a396a24fe6fe1df42f15d13"},
		}, headers)
}
