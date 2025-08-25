package router

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RouterBuilder provides a clean fluent API for building routes
type RouterBuilder struct {
	subrouter *mux.Router
}

// HandlerFunc represents the JavaScript-like handler signature
type HandlerFunc func(req *Request, res *Response)

// Router creates a new router with the given prefix
func Router(mainRouter *mux.Router, prefix string) *RouterBuilder {
	subrouter := mainRouter.PathPrefix(prefix).Subrouter()
	return &RouterBuilder{
		subrouter: subrouter,
	}
}

// Get adds a GET route
func (r *RouterBuilder) Get(path string, handler HandlerFunc) *RouterBuilder {
	r.subrouter.HandleFunc(path, r.wrapHandler(handler)).Methods("GET")
	return r
}

// Post adds a POST route
func (r *RouterBuilder) Post(path string, handler HandlerFunc) *RouterBuilder {
	r.subrouter.HandleFunc(path, r.wrapHandler(handler)).Methods("POST")
	return r
}

// Put adds a PUT route
func (r *RouterBuilder) Put(path string, handler HandlerFunc) *RouterBuilder {
	r.subrouter.HandleFunc(path, r.wrapHandler(handler)).Methods("PUT")
	return r
}

// Delete adds a DELETE route
func (r *RouterBuilder) Delete(path string, handler HandlerFunc) *RouterBuilder {
	r.subrouter.HandleFunc(path, r.wrapHandler(handler)).Methods("DELETE")
	return r
}

// Patch adds a PATCH route
func (r *RouterBuilder) Patch(path string, handler HandlerFunc) *RouterBuilder {
	r.subrouter.HandleFunc(path, r.wrapHandler(handler)).Methods("PATCH")
	return r
}

// wrapHandler converts HandlerFunc to http.HandlerFunc
func (r *RouterBuilder) wrapHandler(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, httpReq *http.Request) {
		req := NewRequest(httpReq)
		res := NewResponse(w)
		handler(req, res)
	}
}
