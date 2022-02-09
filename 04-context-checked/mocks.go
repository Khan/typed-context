
package main

import (
	"context"
	"fmt"
)

// ================================
// Some mock implementations to support doing the thing
// ================================
func GetContextWithAllTheMocks() context.Context {
	ctx := context.Background()

	ctx = context.WithValue(ctx, "request", &Request{ key: "mockUser"})
	ctx = context.WithValue(ctx, "database", &Database{})
	ctx = context.WithValue(ctx, "httpClient", &HttpClient{})
	ctx = context.WithValue(ctx, "secrets", &Secrets{})
	ctx = context.WithValue(ctx, "logger", &Logger{})

	return ctx
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

func (*Database) Read(ctx context.Context, key DatabaseKey) (*User, error) {
	fmt.Printf("Database Reading %v\n", string(key))
	return &User{name: string(key)}, nil
}

type Secrets struct{}

type HttpClient struct {}

func (*HttpClient) Post(ctx context.Context, url string, param string) error {
	fmt.Printf("HTTP Posting %v?%v\n", url, param)
	return nil
}

type Logger struct {}