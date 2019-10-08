package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	port     int
	certPath string
	keyPath  string
	handler  func(w http.ResponseWriter, r *http.Request)
}

func NewServer(port int, certPath string, keyPath string, handler func(w http.ResponseWriter, r *http.Request)) *Server {
	return &Server{
		port:     port,
		certPath: certPath,
		keyPath:  keyPath,
		handler:  handler,
	}
}

func (s *Server) Run(stopCh <-chan struct{}) {
	var wg sync.WaitGroup

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: http.HandlerFunc(s.handler),
		TLSConfig: &tls.Config{
			NextProtos: []string{"h2", "http/1.1"},
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Serving on https://0.0.0.0:%d", s.port)
		if err := server.ListenAndServeTLS(s.certPath, s.keyPath); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-stopCh
		log.Printf("Shutting down sleep server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		server.Shutdown(ctx)
		cancel()
	}()
	wg.Wait()
}

func (_ *Server) PrintMetric() {
	// Do nothing
}
