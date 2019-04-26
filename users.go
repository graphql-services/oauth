// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/graphql-services/oauth/database"
	"github.com/jinzhu/gorm"
)

type User struct {
	ID                  string        `gorm:"primary_key"`
	Email               string        `json:"email"`
	GivenName           *string       `json:"given_name"`
	FamilyName          *string       `json:"family_name"`
	MiddleName          *string       `json:"middle_name"`
	Nickname            *string       `json:"nickname"`
	PreferredUsername   *string       `json:"preferred_username"`
	Profile             *string       `json:"profile"`
	Picture             *string       `json:"picture"`
	Website             *string       `json:"website"`
	Gender              *UserGender   `json:"gender"`
	Birthdate           *time.Time    `json:"birthdate"`
	Zoneinfo            *string       `json:"zoneinfo"`
	Locale              *string       `json:"locale"`
	PhoneNumber         *string       `json:"phone_number"`
	PhoneNumberVerified *string       `json:"phone_number_verified"`
	Address             *string       `json:"address"`
	UpdatedAt           *time.Time    `json:"updatedAt"`
	CreatedAt           time.Time     `json:"createdAt"`
	Accounts            []UserAccount `gorm:"foreignkey:UserID"`
}
type UserAccount struct {
	ID        string     `gorm:"primary_key"`
	Type      string     `json:"type" gorm:"primary_key"`
	UpdatedAt *time.Time `json:"updatedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	User      User       `json:"user"`
	UserID    string     `json:"user_id"`
}

type UserStore struct {
	db *database.DB
}

func (s *UserStore) Automigrate() error {
	return s.db.AutoMigrate(&User{}, &UserAccount{})
}

func (s *UserStore) GetOrCreateUserWithAccount(accountID, email, accountType string) (user *User, err error) {
	user, err = s.GetUserByAccount(accountID, accountType)
	if err != nil {
		return
	}
	if user == nil {
		tx := s.db.Client().Begin()
		user, err = s.CreateUserWithAccount(tx, accountID, email, accountType)
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit().Error
	}
	return
}

func (s *UserStore) GetUserByAccount(accountID, accountType string) (user *User, err error) {
	var account UserAccount
	res := s.db.Client().Model(&UserAccount{ID: accountID, Type: accountType}).Preload("User").First(&account)
	if res.RecordNotFound() {
		return
	}
	err = res.Error
	if err != nil {
		return
	}
	user = &account.User
	return
}

func (s *UserStore) CreateUserWithAccount(tx *gorm.DB, accountID, email, accountType string) (user *User, err error) {
	user = &User{}

	res := tx.First(user, "email = ?", email)
	err = res.Error
	if err != nil && !res.RecordNotFound() {
		return
	}

	if res.RecordNotFound() {
		user = &User{
			ID:    uuid.New().String(),
			Email: email,
		}
	}

	err = tx.Save(user).Error
	if err != nil {
		return
	}

	err = tx.Model(user).Association("Accounts").Append(UserAccount{Type: accountType, ID: accountID}).Error

	return
}
