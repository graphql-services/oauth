package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	errs "errors"

	"github.com/dgrijalva/jwt-go"
	opentracing "github.com/opentracing/opentracing-go"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/utils/uuid"
)

type JWTUser struct {
	Email string `json:"email"`
}

// JWTAccessClaims jwt claims
type JWTAccessClaims struct {
	Scope string  `json:"scope"`
	User  JWTUser `json:"user"`
	jwt.StandardClaims
}

// Valid claims verification
func (a *JWTAccessClaims) Valid() error {
	if time.Unix(a.ExpiresAt, 0).Before(time.Now()) {
		return errors.ErrInvalidAccessToken
	}
	return nil
}

// NewJWTAccessGenerate create to generate the jwt access token instance
func NewJWTAccessGenerate(method jwt.SigningMethod, userStore *UserStore) *JWTAccessGenerate {
	return &JWTAccessGenerate{
		SignedMethod: method,
		UserStore:    userStore,
	}
}

// JWTAccessGenerate generate the jwt access token
type JWTAccessGenerate struct {
	SignedMethod jwt.SigningMethod
	UserStore    *UserStore
}

// Token based on the UUID generated token
func (a *JWTAccessGenerate) Token(data *oauth2.GenerateBasic, isGenRefresh bool) (access, refresh string, err error) {
	ctx := context.Background()

	spanName := "/token/create"
	if isGenRefresh {
		spanName = "/token/refresh"
	}
	span, ctx := opentracing.StartSpanFromContext(ctx, "oauth - "+spanName)
	defer span.Finish()

	scope := data.Request.FormValue("scope")

	fmt.Println("feching user", data.UserID)
	user, fetchErr := a.UserStore.GetUser(ctx, data.UserID)
	err = fetchErr
	if err != nil {
		return
	}

	standardScopes, nonstandardScopes := separateScopes(scope)
	if len(nonstandardScopes) > 0 {
		fmt.Println("validating nonstandard scopes", nonstandardScopes, data.UserID)
		validatedScopes, err := validateScopeForUser(ctx, strings.Join(nonstandardScopes, " "), data.UserID)
		if err != nil {
			return access, refresh, err
		}
		nonstandardScopes = strings.Split(validatedScopes, " ")
	}

	scope = strings.Join(append(standardScopes, nonstandardScopes...), " ")

	jwtUser := JWTUser{}
	if user != nil {
		jwtUser.Email = user.Email
	}

	claims := &JWTAccessClaims{
		StandardClaims: jwt.StandardClaims{
			Audience:  data.Client.GetID(),
			Subject:   data.UserID,
			ExpiresAt: data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix(),
		},
		Scope: scope,
		User:  jwtUser,
	}

	token := jwt.NewWithClaims(a.SignedMethod, claims)
	signedKey, kid, err := getRSAKey()
	if err != nil {
		return
	}

	token.Header["kid"] = kid

	var key interface{}
	if a.isEs() {
		key, err = jwt.ParseECPrivateKeyFromPEM(signedKey.D.Bytes())
		if err != nil {
			return "", "", err
		}
	} else if a.isRsOrPS() {
		key = signedKey
	} else if a.isHs() {
		key = signedKey
	} else {
		return "", "", errs.New("unsupported sign method")
	}
	access, err = token.SignedString(key)
	if err != nil {
		return
	}

	if isGenRefresh {
		refresh = base64.URLEncoding.EncodeToString(uuid.NewSHA1(uuid.Must(uuid.NewRandom()), []byte(access)).Bytes())
		refresh = strings.ToUpper(strings.TrimRight(refresh, "="))
	}

	return
}

func (a *JWTAccessGenerate) isEs() bool {
	return strings.HasPrefix(a.SignedMethod.Alg(), "ES")
}

func (a *JWTAccessGenerate) isRsOrPS() bool {
	isRs := strings.HasPrefix(a.SignedMethod.Alg(), "RS")
	isPs := strings.HasPrefix(a.SignedMethod.Alg(), "PS")
	return isRs || isPs
}

func (a *JWTAccessGenerate) isHs() bool {
	return strings.HasPrefix(a.SignedMethod.Alg(), "HS")
}

func containsScope(scopes, s string) bool {
	_scopes := strings.Split(scopes, " ")
	for _, _s := range _scopes {
		if _s == s {
			return true
		}
	}
	return false
}

func separateScopes(scopes string) (standard, nonstandard []string) {
	standard = []string{}
	nonstandard = []string{}
	if scopes == "" {
		return
	}
	for _, scope := range strings.Split(scopes, " ") {
		if scope == "openid" || scope == "profile" || scope == "email" {
			standard = append(standard, scope)
		} else {
			nonstandard = append(nonstandard, scope)
		}
	}
	return
}
