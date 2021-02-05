package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
	"golang.org/x/exp/errors/fmt"
	"gopkg.in/yaml.v2"
)

// Server ...
type Server struct {
	Router      *mux.Router
	Switchboard map[string]int
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

// ServeReverseProxy ...
func (s *Server) ServeReverseProxy(url *url.URL, w http.ResponseWriter, r *http.Request) {
	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme

	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host
	proxy.ServeHTTP(w, r)
}

// LoadConfig ...
func (s *Server) LoadConfig() error {
	yamlFile, err := ioutil.ReadFile("switchboard.yaml")
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlFile, &s.Switchboard)
}

// RegisterRoutes ...
func (s *Server) RegisterRoutes() {
	s.Router.PathPrefix("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			url := r.URL

			url.Scheme = "http"
			url.Host = fmt.Sprintf("localhost:%d", s.Switchboard[url.Host])

			s.ServeReverseProxy(url, w, r)
		},
	).Methods("OPTIONS", "POST")
}

func main() {
	server := Server{
		Router: mux.NewRouter(),
	}

	err := server.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	server.RegisterRoutes()
}
