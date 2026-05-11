package server

import (
	"fmt"
	"net/http"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	mux *http.ServeMux
}

func New() *Server {
	return &Server{
		mux: http.NewServeMux(),
	}
}

func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) GET(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodGet, addr, handlerFunc, middlewares...)
}

func (s *Server) POST(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodPost, addr, handlerFunc, middlewares...)
}

func (s *Server) PUT(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodPut, addr, handlerFunc, middlewares...)
}

func (s *Server) PATCH(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodPatch, addr, handlerFunc, middlewares...)
}

func (s *Server) DELETE(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodDelete, addr, handlerFunc, middlewares...)
}

func (s *Server) OPTIONS(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodOptions, addr, handlerFunc, middlewares...)
}

func (s *Server) HEAD(addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	s.handle(http.MethodHead, addr, handlerFunc, middlewares...)
}

func (s *Server) handle(method string, addr string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	var handler http.Handler = handlerFunc

	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	s.mux.Handle(fmt.Sprintf("%s %s", method, addr), handler)
}
