// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"context"
	"time"

	"github.com/graphql-services/oauth/database"
	opentracing "github.com/opentracing/opentracing-go"
)

type UserAccount struct {
	ID        string     `gorm:"primary_key"`
	Type      string     `json:"type" gorm:"primary_key"`
	UpdatedAt *time.Time `json:"updatedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UserID    string     `json:"user_id"`
}

type UserStore struct {
	DB *database.DB
	ID *IDClient
}

func (s *UserStore) AutoMigrate() error {
	return s.DB.AutoMigrate(&UserAccount{})
}

func (s *UserStore) GetOrCreateUserWithAccount(ctx context.Context, accountID, email, accountType string) (user *User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetOrCreateUserWithAccount")
	defer span.Finish()

	user, err = s.GetUserByAccount(ctx, accountID, accountType)
	if err != nil {
		return
	}

	if user == nil {
		user, err = s.CreateUserWithAccount(ctx, accountID, email, accountType)
		if err != nil {
			return
		}
	}
	return
}

func (s *UserStore) GetUser(ctx context.Context, userID string) (user *User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetUser")
	defer span.Finish()

	user, err = s.ID.GetUser(ctx, userID)
	return
}

func (s *UserStore) GetUserByAccount(ctx context.Context, accountID, accountType string) (user *User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetUserByAccount")
	defer span.Finish()

	var account UserAccount
	res := s.DB.Client().Model(&UserAccount{}).First(&account, &UserAccount{ID: accountID, Type: accountType})
	if res.RecordNotFound() {
		return
	}
	err = res.Error
	if err != nil {
		return
	}
	user, err = s.ID.GetUser(ctx, account.UserID)
	return
}

func (s *UserStore) CreateUserWithAccount(ctx context.Context, accountID, email, accountType string) (user *User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CreateUserWithAccount")
	defer span.Finish()

	user, err = s.ID.InviteUser(ctx, email)
	if err != nil {
		return
	}

	account := UserAccount{Type: accountType, ID: accountID, UserID: user.ID}
	err = s.DB.Client().FirstOrCreate(&account, account).Error

	return
}
