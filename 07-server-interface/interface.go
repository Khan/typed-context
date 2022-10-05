package main

import "context"

type RequestServer interface {
	Request() *Request
}

type DatabaseInterface interface {
	Read(
		ctx context.Context,
		server interface {
			SecretsServer
			LoggerServer
		},
		key DatabaseKey,
	) (*User, error)
}

type DatabaseServer interface {
	Database() DatabaseInterface
}

type HttpClientServer interface {
	HttpClient() *HttpClient
}

type SecretsServer interface {
	Secrets() *Secrets
}

type LoggerServer interface {
	Logger() *Logger
}
