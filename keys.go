// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/lestrrat/go-jwx/jwk"
	"github.com/patrickmn/go-cache"
)

type RSAKey struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

var c *cache.Cache

// get RSA Key with caching
func getRSAKey() (key *rsa.PrivateKey, err error) {
	if c == nil {
		c = cache.New(5*time.Minute, 10*time.Minute)
	}
	v, ok := c.Get("rsaKey")
	if ok {
		key = v.(*rsa.PrivateKey)
		return
	}

	key, err = fetchRSAKey()
	if err == nil {
		c.Set("rsaKey", key, cache.DefaultExpiration)
	}
	return
}

func fetchRSAKey() (key *rsa.PrivateKey, err error) {
	providerURL := os.Getenv("JWKS_PROVIDER_URL")
	if providerURL == "" {
		err = fmt.Errorf("Missing JWKS_PROVIDER_URL environment variable")
		return
	}
	set, err := jwk.Fetch(providerURL)
	if err != nil {
		err = fmt.Errorf("failed to lookup key: %s", err)
		return
	}

	// If you KNOW you have exactly one key, you can just
	// use set.Keys[0]
	keys := set.Keys
	if len(keys) == 0 {
		err = fmt.Errorf("failed to lookup key: %s", err)
		return
	}

	_key, err := keys[0].Materialize()
	if err != nil {
		return
	}

	key, ok := _key.(*rsa.PrivateKey)
	if !ok {
		err = fmt.Errorf("Cannot convert key to RSA Private Key")
	}
	return
}
