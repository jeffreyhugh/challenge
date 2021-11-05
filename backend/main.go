package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/qbxt/gologger"
	"github.com/sirupsen/logrus"
	"queue.bot/challenge/handlers"
	"queue.bot/challenge/structs"
)

func main() {
	gologger.Init(logrus.DebugLevel)

	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		gologger.Fatal("could not make docker client", err, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cr := structs.CustomRouter{
		Router: mux.NewRouter(),
		Docker: docker,
	}

	cr.HandleFunc("/run/{language}", handlers.HandleRun)

	srv := &http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: cr.Router,
	}

	gologger.Info("server is running", nil)
	go func() {
		if err := srv.ListenAndServeTLS("./certs/cert.pem", "./certs/key.pem"); err != nil {
			gologger.Error("server error", err, nil)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	_ = srv.Shutdown(ctx)

	gologger.Info("received interrupt, shutting down", nil)
}
