package main

import (
	"context"
	"net/http"

	"bookworm.onatim.com/internal/data"
)

// Define a custom contextKey type for context operations.
type contextKey string

// Define a context key for user operations.
const userContextKey = contextKey("user")

// Returns a new copy of the request with the provided User struct added to the context.
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// Retrieves the User struct from the request context.
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
