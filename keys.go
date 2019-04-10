// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"

	"gopkg.in/square/go-jose.v2"
)

type RSAKey struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func generateNewRSAKey() (key *rsa.PrivateKey, err error) {
	reader := rand.Reader
	bitSize := 1024

	key, err = rsa.GenerateKey(reader, bitSize)

	pemKey, err := getPublicPEMKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	fmt.Println("Generated new key with public key:\n", string(pemKey))

	return
}

func getPEMKey(key *rsa.PrivateKey) ([]byte, error) {
	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	buf := new(bytes.Buffer)
	err := pem.Encode(buf, privateKey)
	return buf.Bytes(), err
}

func getPublicPEMKey(key *rsa.PublicKey) ([]byte, error) {
	ans1n, _ := x509.MarshalPKIXPublicKey(key)

	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: ans1n,
	}

	buf := new(bytes.Buffer)
	err := pem.Encode(buf, pemkey)
	return buf.Bytes(), err
}

func jwksHandler(key *rsa.PrivateKey, keyID string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		set := jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				jose.JSONWebKey{
					Algorithm: "RS256",
					Use:       "sig",
					Key:       &key.PublicKey,
					KeyID:     keyID,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(set)
	}
}
