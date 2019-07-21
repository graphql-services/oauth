package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/errors"
)

type IDTokenProfileClaims struct {
	Name              *string    `json:"name,omitempty"`
	FamilyName        *string    `json:"family_name,omitempty"`
	GivenName         *string    `json:"given_name,omitempty"`
	MiddleName        *string    `json:"middle_name,omitempty"`
	Nickname          *string    `json:"nickname,omitempty"`
	PreferredUsername *string    `json:"preferred_username,omitempty"`
	Profile           *string    `json:"profile,omitempty"`
	Picture           *string    `json:"picture,omitempty"`
	Website           *string    `json:"website,omitempty"`
	Gender            *string    `json:"gender,omitempty"`
	Birthdate         *time.Time `json:"birthdate,omitempty"`
	Zoneinfo          *string    `json:"zoneinfo,omitempty"`
	Locale            *string    `json:"locale,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}
type IDTokenEmailClaims struct {
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

type IDTokenClaims struct {
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Issuer    string `json:"iss,omitempty"`
	Subject   string `json:"sub,omitempty"`
	IDTokenEmailClaims
	IDTokenProfileClaims
}

func generateIDToken(ctx context.Context, ti oauth2.TokenInfo, us *UserStore) (token string, err error) {
	user, err := us.GetUser(ctx, ti.GetUserID())
	scope := ti.GetScope()

	claims := &IDTokenClaims{
		Audience:  ti.GetClientID(),
		Subject:   user.ID,
		ExpiresAt: ti.GetAccessCreateAt().Add(ti.GetAccessExpiresIn()).Unix(),
	}

	if containsScope(scope, "email") {
		claims.IDTokenEmailClaims = IDTokenEmailClaims{
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
		}
	}
	if containsScope(scope, "profile") {
		name := ""
		if user.FamilyName != nil && user.GivenName != nil {
			name = fmt.Sprintf("%s %s", *user.GivenName, *user.FamilyName)
		}
		claims.IDTokenProfileClaims = IDTokenProfileClaims{
			Name:              &name,
			FamilyName:        user.FamilyName,
			GivenName:         user.GivenName,
			MiddleName:        user.MiddleName,
			Nickname:          user.Nickname,
			PreferredUsername: user.PreferredUsername,
			Profile:           user.Profile,
			Picture:           user.Picture,
			Website:           user.Website,
			Gender:            user.Gender,
			Birthdate:         user.Birthdate,
			Zoneinfo:          user.Zoneinfo,
			Locale:            user.Locale,
			UpdatedAt:         user.UpdatedAt,
		}
	}

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedKey, err := getRSAKey()
	if err != nil {
		return
	}
	token, err = t.SignedString(signedKey)

	return
}

// Valid claims verification
func (a *IDTokenClaims) Valid() error {
	if time.Unix(a.ExpiresAt, 0).Before(time.Now()) {
		return errors.ErrInvalidAccessToken
	}
	return nil
}
