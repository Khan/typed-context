package main

import (
	"context"
	"fmt"
)

// ================================
// Some mock implementations to support doing the thing
// ================================
func GetServerWithAllTheMocks() MockServer {
	return MockServer{
		request:    &Request{key: "mockUser"},
		database:   &Database{},
		httpClient: &HttpClient{},
		secrets:    &Secrets{},
		logger:     &Logger{},
	}
}

type MockServer struct {
	request    *Request
	database   *Database
	httpClient *HttpClient
	secrets    *Secrets
	logger     *Logger
}

func (c MockServer) Request() *Request {
	return c.request
}

func (c MockServer) Database() DatabaseInterface {
	return c.database
}

func (c MockServer) HttpClient() *HttpClient {
	return c.httpClient
}

func (c MockServer) Secrets() *Secrets {
	return c.secrets
}

func (c MockServer) Logger() *Logger {
	return c.logger
}

type Request struct {
	key DatabaseKey
}

func (r *Request) GetUserKey() (DatabaseKey, error) {
	fmt.Printf("Request getting key %v\n", r.key)
	return r.key, nil
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
	ctx context.Context,
	server interface {
		SecretsServer
		LoggerServer
	},
	key DatabaseKey,
) (*User, error) {
	fmt.Printf("Database Reading %v\n", string(key))
	return &User{name: string(key)}, nil
}

type Secrets struct{}

type HttpClient struct{}

func (*HttpClient) Post(
	ctx context.Context,
	server interface {
		RequestServer
	},
	url string,
	param string,
) error {
	fmt.Printf("HTTP Posting %v?%v\n", url, param)
	return nil
}

type Logger struct{}
