
package main

func DoTheThing(thing string) error {
	// Find User Key from request
	userKey, err := request.GetUserKey()
	if err != nil { return err }

	// Lookup User in database
	user, err := database.Read(userKey)
	if err != nil { return err }

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		err = httpClient.Post("www.dothething.example", user.GetName())
	}
	return err
}

func main() {
	_ = DoTheThing("a thing")
}
