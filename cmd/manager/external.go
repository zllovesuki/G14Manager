package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/zllovesuki/G14Manager/system/shared"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	suture "github.com/thejerf/suture/v4"
)

type externalWeb struct {
	srv *http.Server
	s   *grpcweb.WrappedGrpcServer
}

func NewWeb(s *grpcweb.WrappedGrpcServer) *externalWeb {
	return &externalWeb{
		srv: &http.Server{
			Addr: shared.WebAddress,
		},
		s: s,
	}
}

func (g *externalWeb) String() string {
	return "externalWeb"
}

func (g *externalWeb) Serve(haltCtx context.Context) error {
	errCh := make(chan error)
	mux := http.NewServeMux()
	mux.Handle("/debug/logs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsDebug != "no" {
			fmt.Fprintf(w, "Logging is not enabled on debug build")
			return
		}
		osFile, err := os.Open(logLocation)
		if err != nil {
			fmt.Fprintf(w, "Unable to open log file: %+v", err)
			return
		}
		defer osFile.Close()
		io.Copy(w, osFile)
	}))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-user-agent, x-grpc-web, grpc-status, grpc-message")
		w.Header().Set("Access-Control-Expose-Headers", "grpc-status, grpc-message")
		if r.Method == "OPTIONS" {
			return
		}
		if g.s.IsGrpcWebRequest(r) {
			g.s.ServeHTTP(w, r)
		} else {
			http.DefaultServeMux.ServeHTTP(w, r)
		}
	}))

	g.srv.Handler = mux

	go func() {
		log.Printf("[externalWeb] externalWeb available at %s\n", g.srv.Addr)
		errCh <- g.srv.ListenAndServe()
	}()
	for {
		select {
		case <-haltCtx.Done():
			log.Println("[externalWeb] exiting externalWeb server")
			g.srv.Shutdown(context.Background())
			return nil
		case err := <-errCh:
			if err == nil || err == http.ErrServerClosed {
				return nil
			}
			log.Printf("[externalWeb] error channel: %s\n", err)
			return suture.ErrTerminateSupervisorTree
		}
	}
}
