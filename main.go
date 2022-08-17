package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v2"
)

// Server ...
type Server struct {
	http.Server
	Switchboard map[string]int
}

func newServer(cm *autocert.Manager) (*Server, error) {
	var server Server
	err := server.LoadConfig()
	if err != nil {
		return &server, err
	}

	server.RegisterRoutes()
	server.Addr = ":https"

	server.TLSConfig = &tls.Config{
		GetCertificate: cm.GetCertificate,
	}

	return &server, nil
}

// ServeReverseProxy ...
func (s *Server) ServeReverseProxy(port int, w http.ResponseWriter, r *http.Request) {
	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
		Path:   "/",
	})

	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
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
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		port := s.Switchboard[strings.TrimPrefix(r.Host, "www.")]
		s.ServeReverseProxy(port, w, r)
	})

	s.Handler = mux
}

func main() {
	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache("certs"),
	}

	server, err := newServer(&certManager)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go http.ListenAndServe(":80", certManager.HTTPHandler(nil))
	log.Fatalf("Failed to start server: %v", server.ListenAndServeTLS("", ""))
}
