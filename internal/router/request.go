package router

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
)

// Type aliases for cleaner syntax
type Req = Request
type Res = Response

// Request provides a clean interface for handling HTTP requests (like Express.js req)
type Request struct {
	*http.Request
	Vars  map[string]string // URL path variables
	Query url.Values        // Query parameters
}

// NewRequest creates a new request wrapper
func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
		Vars:    mux.Vars(r),
		Query:   r.URL.Query(),
	}
}

// JSON parses the request body as JSON into the provided struct
func (req *Request) JSON(v interface{}) error {
	return json.NewDecoder(req.Body).Decode(v)
}

// Param gets a URL path variable by name
func (req *Request) Param(name string) string {
	return req.Vars[name]
}

// QueryParam gets a query parameter by name
func (req *Request) QueryParam(name string) string {
	return req.Query.Get(name)
}

// QueryInt gets a query parameter as integer
func (req *Request) QueryInt(name string, defaultValue int) int {
	value := req.Query.Get(name)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// QueryBool gets a query parameter as boolean
func (req *Request) QueryBool(name string, defaultValue bool) bool {
	value := req.Query.Get(name)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}

// GetHeader gets a request header by name (alias for easier access)
func (req *Request) GetHeader(name string) string {
	return req.Header.Get(name)
}
