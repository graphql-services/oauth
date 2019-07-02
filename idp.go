// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/machinebox/graphql"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	fetchIDPUserQuery = `
query($email: String!, $password: String!) {
	result: login(email: $email, password: $password) {
		id
		email
		email_verified
	}
}  
`
	createIDPUserMutation = `
mutation($email: String!, $password: String!) {
	result: createUser(input: {email:$email, password: $password}) {
		id
		email
		email_verified
	}
}  
`
)

type IDPClient struct {
	URL string
}

func NewIDPClient() *IDPClient {
	URL := os.Getenv("IDP_URL")

	if URL == "" {
		panic(fmt.Errorf("Missing required environment variable IDP_URL"))
	}
	return &IDPClient{URL}
}

type IDPUser struct {
	ID            string
	Email         string
	EmailVerified bool `json:"email_verified"`
}
type IDPUserResponse struct {
	Result IDPUser
}

func (c *IDPClient) FetchIDPUser(ctx context.Context, email, password string) (user IDPUser, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FetchIDPUser")
	defer span.Finish()

	var res IDPUserResponse

	req := graphql.NewRequest(fetchIDPUserQuery)
	req.Var("email", email)
	req.Var("password", password)
	err = c.sendRequest(ctx, req, &res)

	user = res.Result

	return
}

func (c *IDPClient) CreateIDPUser(ctx context.Context, email, password string) (user IDPUser, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CreateIDPUser")
	defer span.Finish()

	var res IDPUserResponse

	req := graphql.NewRequest(createIDPUserMutation)
	req.Var("email", email)
	req.Var("password", password)
	err = c.sendRequest(ctx, req, &res)

	user = res.Result

	return
}

func (c *IDPClient) sendRequest(ctx context.Context, req *graphql.Request, data interface{}) error {
	client := graphql.NewClient(c.URL)
	client.Log = func(s string) {
		log.Println(s)
	}

	return client.Run(ctx, req, data)
}
