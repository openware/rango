package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
	"os"

	"github.com/dgrijalva/jwt-go"
)

type KeyStore struct {
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func LoadOrGenerateKeys(privPath, pubPath string) (*KeyStore, error) {
	var err error
	ks := &KeyStore{}

	if fileExist(privPath) {
		if err = ks.LoadPrivateKey(privPath); err != nil {
			return ks, err
		}
	} else {
		ks.GenerateKeys()
		if err = ks.SavePrivateKey(privPath); err != nil {
			return ks, err
		}
	}

	if fileExist(pubPath) {
		if err = ks.LoadPublicKeyFromFile(pubPath); err != nil {
			return ks, err
		}
	} else {
		if ks.PublicKey == nil {
			ks.PublicKey = &ks.PrivateKey.PublicKey
		}

		if err = ks.SavePublicKey(pubPath); err != nil {
			return ks, err
		}
	}

	return ks, nil
}

func (ks *KeyStore) LoadPublicKeyFromFile(path string) error {
	pem, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return err
	}

	ks.PublicKey = key

	return nil
}

func (ks *KeyStore) LoadPublicKeyFromString(str string) error {
	pem, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return err
	}

	ks.PublicKey = key
	return nil
}

func (ks *KeyStore) LoadPrivateKey(path string) error {
	pem, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		return err
	}

	ks.PrivateKey = key
	return nil
}

func (ks *KeyStore) GenerateKeys() error {
	reader := rand.Reader
	bitSize := 2048
	key, err := rsa.GenerateKey(reader, bitSize)

	if err != nil {
		return err
	}

	ks.PrivateKey = key
	ks.PublicKey = &key.PublicKey

	return nil
}

func (ks *KeyStore) SavePrivateKey(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	var key = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(ks.PrivateKey),
	}

	return pem.Encode(file, key)
}

func (ks *KeyStore) SavePublicKey(path string) error {
	bytes, err := x509.MarshalPKIXPublicKey(ks.PublicKey)
	if err != nil {
		return err
	}

	var key = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: bytes,
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	return pem.Encode(file, key)
}
