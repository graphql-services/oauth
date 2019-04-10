package main

import (
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"github.com/lestrrat/go-jwx/jwk"
)

func testJWKS() {
	time.Sleep(time.Second * 3)
	set, err := jwk.Fetch("http://localhost:8080/.well-known/jwks.json")
	if err != nil {
		log.Printf("failed to parse JWK: %s", err)
		return
	}

	// If you KNOW you have exactly one key, you can just
	// use set.Keys[0]
	keys := set.Keys
	if len(keys) == 0 {
		log.Printf("failed to lookup key: %s", err)
		return
	}

	key, err := keys[0].Materialize()
	if err != nil {
		log.Printf("failed to create public key: %s", err)
		return
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	pem, _ := getPublicPEMKey(rsaKey)
	fmt.Println("???", ok, string(pem))
}
