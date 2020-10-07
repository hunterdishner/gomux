package gomux

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hunterdishner/errors"
	"github.com/phayes/freeport"
	"github.com/rs/cors"
)

type Server struct {
	name      string
	ctx       context.Context
	mux       *mux.Router
	tls       bool
	port      int
	tlsconfig *tls.Config
	cors      *cors.Cors
}

type ServiceHandler func(io.Writer, *http.Request) (interface{}, error)

type Option func(*Server)

func TLS() Option {
	return func(s *Server) {
		s.tls = true
	}
}

func TLSConfig(conf *tls.Config) Option {
	return func(s *Server) {
		s.tlsconfig = conf
		s.tls = true
	}
}

func CustomCors(cors *cors.Cors) Option {
	return func(s *Server) {
		s.cors = cors
	}
}

func Port(p int) Option {
	return func(s *Server) {
		s.port = p
	}
}

func New(ctx context.Context, name string, opts ...Option) *Server {
	s := &Server{
		name: name,
		mux:  mux.NewRouter().StrictSlash(true).PathPrefix("/" + name).Subrouter(),
		ctx:  ctx,
		tlsconfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
		cors: cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
		}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Route struct {
	Method      string
	Path        string
	Handler     ServiceHandler
	HandlerFunc http.HandlerFunc
}

// NewRoute is a convenience function to make calling AddRoutes simpler.
func NewRoute(method, path string, handler ServiceHandler) Route {
	return Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	}
}

// NewRouteFn is a convenience function to make calling AddRoutes simpler.
func NewRouteFn(method, path string, handler http.HandlerFunc) Route {
	return Route{
		Method:      method,
		Path:        path,
		HandlerFunc: handler,
	}
}

// Get is a convenience function for creating a route with the GET method.
func Get(path string, handler ServiceHandler) Route {
	return Route{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	}
}

// GetFn is a convenience function for creating a route with the GET method.
func GetFn(path string, handler http.HandlerFunc) Route {
	return Route{
		Method:      "GET",
		Path:        path,
		HandlerFunc: handler,
	}
}

// Post is a convenience function for creating a route with the POST method.
func Post(path string, handler ServiceHandler) Route {
	return Route{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	}
}

// PostFn is a convenience function for creating a route with the POST method.
func PostFn(path string, handler http.HandlerFunc) Route {
	return Route{
		Method:      "POST",
		Path:        path,
		HandlerFunc: handler,
	}
}

// Delete is a convenience function for creating a route with the DELETE method.
func Delete(path string, handler ServiceHandler) Route {
	return Route{
		Method:  "DELETE",
		Path:    path,
		Handler: handler,
	}
}

// DeleteFn is a convenience function for creating a route with the DELETE method.
func DeleteFn(path string, handler http.HandlerFunc) Route {
	return Route{
		Method:      "DELETE",
		Path:        path,
		HandlerFunc: handler,
	}
}

// Delete is a convenience function for creating a route with the PUT method.
func Put(path string, handler ServiceHandler) Route {
	return Route{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	}
}

// DeleteFn is a convenience function for creating a route with the PUT method.
func PutFn(path string, handler http.HandlerFunc) Route {
	return Route{
		Method:      "PUT",
		Path:        path,
		HandlerFunc: handler,
	}
}

func (s *Server) AddRoutes(routes ...Route) *Server {
	for _, route := range routes {
		route.Path = "/" + strings.TrimPrefix(route.Path, "/")
		if route.Handler != nil {
			route.HandlerFunc = s.responseHandler(route.Handler)
		}

		if err := s.mux.Methods(route.Method).Path(route.Path).HandlerFunc(route.HandlerFunc).GetError(); err != nil { //goes against how go does things but it works for this case and is relatively legible
			//log error
			log.Printf("%+v", errors.E(errors.Invalid, errors.Code(http.StatusUnprocessableEntity), err))
		}
	}

	return s
}

func (s *Server) Serve() error {
	if s.port == 0 {
		free, err := freeport.GetFreePort()
		if err != nil {
			return errors.E(errors.CodeServerError, errors.HTTP, err)
		}
		s.port = free
	}

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(s.port),
		Handler:      s.cors.Handler(s.mux),
		TLSConfig:    s.tlsconfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Printf("\n%s started on port %d\n", s.name, s.port)
	if s.tls {
		return srv.ListenAndServeTLS("server.crt", "server.key")
	}

	return srv.ListenAndServe()
}

func (s *Server) responseHandler(fn ServiceHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		data, err := fn(w, r)
		if err != nil {
			switch err := err.(type) {
			case *errors.Error:
				if err.Code == 0 {
					err.Code = http.StatusInternalServerError
				}
				w.WriteHeader(int(err.Code))
				data = err
			default:
				w.WriteHeader(int(errors.CodeServerError))
				data = errors.E(errors.CodeServerError, errors.Invalid, err)
			}
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if err := writeContent(w, r, data); err != nil {
			log.Printf("%+v", errors.E(errors.Invalid, errors.Code(http.StatusUnprocessableEntity), err))
		}
	}
}

func writeContent(w http.ResponseWriter, r *http.Request, data interface{}) error {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return errors.E(errors.Encoding, errors.CodeServerError, err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return errors.E(errors.IO, errors.CodeServerError, err)
	}

	return nil
}
