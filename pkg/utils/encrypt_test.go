package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestEncrypt(t *testing.T) {
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
		return
	}
	enc := Encrypter{pkey}
	c, err := enc.Encrypt([]byte("lorem ipsum"))
	if err != nil {
		t.Error(err)
		return
	}
	d, _ := enc.Decrypt(c)
	if string(d) != "lorem ipsum" {
		t.Errorf("Got %s", string(d))
	}
}

func TestEncryptString(t *testing.T) {
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
		return
	}
	enc := Encrypter{pkey}
	c, err := enc.EncryptString("lorem ipsum")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(c)
	d, _ := enc.DecryptString(c)
	if d != "lorem ipsum" {
		t.Errorf("Got %s", d)
	}
}
