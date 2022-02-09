
package main

func DoTheThing(
	thing string,
	request *Request,
	database *Database,
	httpClient *HttpClient,
	secrets *Secrets,
	logger *Logger,
	timeout *Timeout,
) error {
	// Find User Key from request
	userKey, err := request.GetUserKey()
	if err != nil { return err }

	// Lookup User in database
	user, err := database.Read(userKey, secrets, logger, timeout)
	if err != nil { return err }

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		token, err := request.GetToken()
		if err != nil { return err }

		err = httpClient.Post("www.dothething.example", user.GetName(), token)
		return err
	}
	return nil
}

func main() {
	_ = DoTheThing(
		"a thing",
		GetMockRequest(),
		GetMockDatabase(),
		GetMockHttpClient(),
		GetMockSecrets(),
		GetMockLogger(),
		GetMockTimeout(),
	)
}
