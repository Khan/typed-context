package main

import (
	"context"
)

func DoTheThing(
	ctx context.Context,
	server interface {
		RequestServer
		DatabaseServer
		HttpClientServer
		SecretsServer
		LoggerServer
	},
	thing string,
) error {
	// Find User Key from request
	userKey, err := server.Request().GetUserKey()
	if err != nil {
		return err
	}

	// Lookup User in database
	user, err := server.Database().Read(ctx, server, userKey)
	if err != nil {
		return err
	}

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		err = server.HttpClient().Post(ctx, server, "www.dothething.example", user.GetName())
	}
	return err
}

func main() {
	ctx := context.Background()
	_ = DoTheThing(
		ctx,
		GetServerWithAllTheMocks(),
		"a thing",
	)
}
