package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// APIKeyHMAC contains API keys credential to authenticate to HTTP API using HMAC
type APIKeyHMAC struct {
	AccessKey string
	SecretKey string
}

// NewAPIKeyHMAC creates an instance of APIKeyHMAC
func NewAPIKeyHMAC(accessKey, secretKey string) *APIKeyHMAC {
	return &APIKeyHMAC{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
}

// GetSignature return a signature for the given nonce, if nonce is zero it use the current time in millisecond
func (key *APIKeyHMAC) GetSignature(nonce int64) string {
	if nonce == 0 {
		nonce = int64(time.Now().UnixNano() * 1000000)
	}
	mac := hmac.New(sha256.New, []byte(key.SecretKey))
	mac.Write([]byte(fmt.Sprintf("%d%s", nonce, key.AccessKey)))
	return hex.EncodeToString(mac.Sum(nil))
}

// GetSignedHeader returns a header with valid HMAC authorization fields
func (key *APIKeyHMAC) GetSignedHeader(nonce int64) http.Header {
	if nonce == 0 {
		nonce = int64(time.Now().UnixNano() * 1000000)
	}

	return http.Header{
		"X-Auth-Apikey":    {key.AccessKey},
		"X-Auth-Nonce":     {fmt.Sprintf("%d", nonce)},
		"X-Auth-Signature": {key.GetSignature(nonce)},
	}
}
