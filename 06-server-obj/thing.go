package main

func DoTheThing(
	thing string,
	server *Server,
) error {
	// Find User Key from request
	userKey, err := server.request.GetUserKey()
	if err != nil {
		return err
	}

	// Lookup User in database
	user, err := server.database.Read(server, userKey)
	if err != nil {
		return err
	}

	// Maybe post an http if can do the thing
	if user.CanDoThing(thing) {
		token, err := server.request.GetToken()
		if err != nil {
			return err
		}

		err = server.httpClient.Post(server, "www.dothething.example", user.GetName(), token)
		return err
	}
	return nil
}

func main() {
	_ = DoTheThing(
		"a thing",
		GetProdServer(),
	)
}
