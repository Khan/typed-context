package main

import "fmt"

// ================================
// Some mock implementations to support doing the thing
// ================================
type Server struct {
	request    *Request
	database   *Database
	httpClient *HttpClient
	secrets    *Secrets
	logger     *Logger
	timeout    *Timeout
}

func GetProdServer() *Server {
	return &Server{
		GetMockRequest(),
		GetMockDatabase(),
		GetMockHttpClient(),
		GetMockSecrets(),
		GetMockLogger(),
		GetMockTimeout(),
	}
}

type Request struct {
	key DatabaseKey
}

func (r *Request) GetUserKey() (DatabaseKey, error) {
	fmt.Printf("Request getting key %v\n", r.key)
	return r.key, nil
}

func GetMockRequest() *Request {
	return &Request{key: "mockUser"}
}

type Token string

func (r *Request) GetToken() (Token, error) {
	return "a Token", nil
}

type User struct {
	name string
}

func (user *User) GetName() string {
	return user.name
}
func (*User) CanDoThing(thing string) bool {
	return true
}

type DatabaseKey string

type Database struct{}

func (*Database) Read(
	server *Server,
	key DatabaseKey,
) (*User, error) {
	fmt.Printf("Database Reading %v\n", string(key))
	return &User{name: string(key)}, nil
}

func GetMockDatabase() *Database {
	return &Database{}
}

type HttpClient struct{}

func (*HttpClient) Post(server *Server, url string, param string, token Token) error {
	fmt.Printf("HTTP Posting %v?%v\n", url, param)
	return nil
}

func GetMockHttpClient() *HttpClient {
	return &HttpClient{}
}

type Secrets struct{}

func GetMockSecrets() *Secrets {
	return &Secrets{}
}

type Logger struct{}

func GetMockLogger() *Logger {
	return &Logger{}
}

type Timeout struct{}

func GetMockTimeout() *Timeout {
	return &Timeout{}
}
