package main

import (
	"context"
	"fmt"
	"os"

	pb "github.com/graphql-services/oauth/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var client *pb.ScopeValidatorClient

func getClient() (*pb.ScopeValidatorClient, error) {
	if client == nil {

		URL := os.Getenv("SCOPE_VALIDATOR_URL")
		if URL != "" {
			conn, err := grpc.Dial(URL, grpc.WithInsecure())
			if err != nil {
				return nil, err
			}

			c := pb.NewScopeValidatorClient(conn)
			client = &c
		}
	}
	return client, nil
}

func validateScopeForUser(ctx context.Context, scope, userID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validateScopeForUser")
	defer span.Finish()

	c, err := getClient()
	if err != nil {
		return "", err
	}
	if c != nil {
		req := &pb.ValidateRequest{UserID: userID, Scopes: scope}
		res, err := (*c).Validate(ctx, req)
		if err != nil {
			return "", err
		}

		if !res.Valid {
			return "", fmt.Errorf("invalid scopes")
		}
		scope = res.Scopes
	}
	return scope, nil
}
