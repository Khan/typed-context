
package main

import "context"

func DoTheThing(
	ctx context.Context,
	thing string,
) error {
	// Find User Key from request
	userKey, err := ctx.Value("request").(*Request).GetUserKey()
	if err != nil { return err }

	// Lookup User in database
	user, err := ctx.Value("database").(*Database).Read(ctx, userKey)
	if err != nil { return err }

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		err = ctx.Value("httpClient").(*HttpClient).
			Post(ctx, "www.dothething.example", user.GetName())
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