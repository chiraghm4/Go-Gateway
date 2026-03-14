package middleware

import (
	"net/http"
)

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// Chain(proxy, A, B, C)
	// A(B(C(proxy)))
	
	for i := len(middlewares) - 1; i >=0 ; i-- {
		h = middlewares[i](h)
	}

	return h
}