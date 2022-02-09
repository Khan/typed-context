
package main

import (
	"context"
	"errors"
)

func DoTheThing(
	ctx context.Context,
	thing string,
) error {
	// Find User Key from request
	request, ok := ctx.Value("request").(*Request)
	if !ok || request == nil { return errors.New("Missing Request") }

	userKey, err := request.GetUserKey()
	if err != nil { return err }

	// Lookup User in database
	database, ok := ctx.Value("database").(*Database)
	if !ok || database == nil { return errors.New("Missing Database") }

	user, err := database.Read(ctx, userKey)
	if err != nil { return err }

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		httpClient, ok := ctx.Value("httpClient").(*HttpClient)
		if !ok || httpClient == nil {
			return errors.New("Missing HttpClient")
		}

		err = httpClient.Post(ctx, "www.dothething.example", user.GetName())
		return err
	}
	return nil
}

func main() {
	ctx := GetContextWithAllTheMocks()
	_ = DoTheThing(
		ctx,
		"a thing",
	)
}