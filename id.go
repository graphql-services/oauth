package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/machinebox/graphql"
)

const (
	getIDUserQuery = `
query($id: ID!) {
	result: user(id: $id) {
		id
		email
	}
}  
`
	inviteIDUserMutation = `
mutation($email: String!) {
	result: inviteUser(email: $email) {
		id
		email
	}
}  
`
)

type User struct {
	ID                  string     `json:"id" gorm:"primary_key"`
	Email               string     `json:"email" gorm:"unique"`
	GivenName           *string    `json:"given_name"`
	FamilyName          *string    `json:"family_name"`
	MiddleName          *string    `json:"middle_name"`
	Nickname            *string    `json:"nickname"`
	PreferredUsername   *string    `json:"preferred_username"`
	Profile             *string    `json:"profile"`
	Picture             *string    `json:"picture"`
	Website             *string    `json:"website"`
	Gender              *string    `json:"gender"`
	Birthdate           *time.Time `json:"birthdate"`
	Zoneinfo            *string    `json:"zoneinfo"`
	Locale              *string    `json:"locale"`
	PhoneNumber         *string    `json:"phone_number"`
	PhoneNumberVerified *string    `json:"phone_number_verified"`
	Address             *string    `json:"address"`
	UpdatedAt           *time.Time `json:"updatedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type IDResponse struct {
	Result *User
}

type IDClient struct {
	URL string
}

func NewIDClient() *IDClient {
	URL := os.Getenv("ID_URL")

	if URL == "" {
		panic(fmt.Errorf("Missing required environment variable ID_URL"))
	}
	return &IDClient{URL}
}

func (c *IDClient) InviteUser(ctx context.Context, email string) (u *User, err error) {
	var res IDResponse

	req := graphql.NewRequest(inviteIDUserMutation)
	req.Var("email", email)
	err = c.sendRequest(ctx, req, &res)

	u = res.Result

	return
}

func (c *IDClient) GetUser(ctx context.Context, id string) (u *User, err error) {
	var res IDResponse

	req := graphql.NewRequest(getIDUserQuery)
	req.Var("id", id)
	err = c.sendRequest(ctx, req, &res)

	u = res.Result

	return
}

func (c *IDClient) sendRequest(ctx context.Context, req *graphql.Request, data interface{}) error {
	client := graphql.NewClient(c.URL)
	client.Log = func(s string) {
		log.Println(s)
	}

	return client.Run(ctx, req, data)
}
