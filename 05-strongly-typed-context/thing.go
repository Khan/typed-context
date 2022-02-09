package main

import (
	"context"
)

func DoTheThing(
	ctx interface {
		context.Context
		RequestContext
		DatabaseContext
		HttpClientContext
		SecretsContext
		LoggerContext
	},
	thing string,
) error {
	// Find User Key from request
	userKey, err := ctx.Request().GetUserKey()
	if err != nil {
		return err
	}

	// Lookup User in database
	user, err := ctx.Database().Read(ctx, userKey)
	if err != nil {
		return err
	}

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		err = ctx.HttpClient().Post(ctx, "www.dothething.example", user.GetName())
	}
	return err
}

func main() {
	ctx := GetContextWithAllTheMocks()
	_ = DoTheThing(
		ctx,
		"a thing",
	)
}
