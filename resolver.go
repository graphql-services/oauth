package main

import (
	"context"

	"github.com/graphql-services/oauth/database"
	uuid "github.com/satori/go.uuid"
	"github.com/sethvargo/go-password/password"
) // THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct {
	DB *database.DB
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) InviteUser(ctx context.Context, email string) (u *User, err error) {
	u = &User{}
	res := r.DB.Client().First(u, "email = ?", email)
	err = res.Error
	if err != nil && !res.RecordNotFound() {
		return
	}

	if res.RecordNotFound() {
		pass, passErr := password.Generate(8, 2, 0, false, false)
		err = passErr
		if err != nil {
			return
		}

		idpUser, idpErr := CreateIDPUser(ctx, email, pass)
		err = idpErr
		if err != nil {
			return
		}

		u = &User{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Email: idpUser.Email,
		}
		err = r.DB.Client().Save(u).Error
	}

	return
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) User(ctx context.Context, id string) (u *User, err error) {
	u = &User{}
	err = r.DB.Client().First(u, "id = ?", id).Error
	return
}
