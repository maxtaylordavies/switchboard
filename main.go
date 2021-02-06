package main

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/exp/errors/fmt"
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
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL

		log.Println(url.Host)

		url.Scheme = "http"
		url.Host = fmt.Sprintf("localhost:%d", s.Switchboard[url.Host])

		s.ServeReverseProxy(url, w, r)
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
