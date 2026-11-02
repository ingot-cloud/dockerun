package main

import (
	"context"
	"dockerun/internal/backup"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/handler"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"dockerun/internal/router"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := config.Initialize(); err != nil {
		log.Fatal(err)
	}
	cfg := config.Load()
	if err := docker.Init(); err != nil {
		log.Fatal(err)
	}
	defer docker.Close()
	if err := database.Init(cfg.DatabasePath); err != nil {
		log.Fatal(err)
	}
	if err := builtin.Init(); err != nil {
		log.Fatal(err)
	}
	ctxInfo, cancelInfo := context.WithTimeout(context.Background(), 5*time.Second)
	if platform, err := release.Platform(ctxInfo); err == nil {
		database.DB.Model(&models.Target{}).Where("id = ?", 1).Update("platform", platform)
	}
	cancelInfo()
	release.Start()
	defer release.Stop()
	backup.Start()
	defer backup.Stop()
	handler.ResumeBundleImports()
	server := &http.Server{Addr: cfg.ServerAddr, Handler: router.Setup(), ReadHeaderTimeout: 15 * time.Second, IdleTimeout: 120 * time.Second}
	errs := make(chan error, 1)
	go func() {
		log.Printf("Dockerun listening on %s", cfg.ServerAddr)
		if cfg.TLSCert != "" {
			errs <- server.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			errs <- server.ListenAndServe()
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case err := <-errs:
		if err != http.ErrServerClosed {
			log.Print(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		server.Close()
	}
}
