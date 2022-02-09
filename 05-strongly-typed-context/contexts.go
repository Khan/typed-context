package main

import "context"

type RequestContext interface {
	Request() *Request
	context.Context
}

type DatabaseInterface interface {
	Read(
		ctx interface{
			context.Context
			SecretsContext
			LoggerContext
		},
		key DatabaseKey,
	) (*User, error)
}

type DatabaseContext interface {
	Database() DatabaseInterface
	context.Context
}

type HttpClientContext interface {
	HttpClient() *HttpClient
	context.Context
}

type SecretsContext interface {
	Secrets() *Secrets
	context.Context
}

type LoggerContext interface {
	Logger() *Logger
	context.Context
}