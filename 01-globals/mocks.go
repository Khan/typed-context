
package main

import "fmt"

// ================================
// Some mock implementations to support doing the thing
// ================================

type Request struct {
	key DatabaseKey
}

func (r *Request) GetUserKey() (DatabaseKey, error) {
	fmt.Printf("Request getting key %v\n", r.key)
	return r.key, nil
}

var request = Request{
	key: "mockUser",
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

func (*Database) Read(key DatabaseKey) (*User, error) {
	fmt.Printf("Database Reading %v\n", string(key))
	return &User{name: string(key)}, nil
}

var database Database

type HttpClient struct {}

func (*HttpClient) Post(url string, param string) error {
	fmt.Printf("HTTP Posting %v?%v\n", url, param)
	return nil
}

var httpClient HttpClient