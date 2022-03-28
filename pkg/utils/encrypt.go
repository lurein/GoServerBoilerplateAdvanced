package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
)

type Encrypter struct {
	PrivateKey *rsa.PrivateKey
}

func Encrypt(in []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, in, []byte(""))
}

func EncryptString(in string, publicKey *rsa.PublicKey) (string, error) {
	s, err := Encrypt([]byte(in), publicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(s), nil
}

func (e Encrypter) Encrypt(in []byte) ([]byte, error) {
	return Encrypt(in, &e.PrivateKey.PublicKey)
}

func (e Encrypter) EncryptString(in string) (string, error) {
	s, err := e.Encrypt([]byte(in))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(s), nil
}

func (e Encrypter) Decrypt(in []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, e.PrivateKey, in, []byte(""))
}

func (e Encrypter) DecryptString(in string) (string, error) {
	s, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return "", err
	}
	c, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, e.PrivateKey, s, []byte(""))
	if err != nil {
		return "", err
	}
	return string(c), nil
}
