package main

import (
	"context"
	log "log/slog"
	"os"
	"os/signal"
	"time"

	"gitlab.com/strongishllama/millhouse.dev/internal/app"

	flag "github.com/spf13/pflag"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.SetDefault(log.New(log.NewJSONHandler(os.Stdout, &log.HandlerOptions{
		Level: log.LevelDebug,
	})))

	if len(os.Args) < 2 {
		log.Error("missing command, expected usage: millhouse <command>")
		os.Exit(1)
	}

	static := os.DirFS("static")
	templates := os.DirFS("templates")
	codeBlocks := os.DirFS("codeBlocks")

	var client app.Client

	switch os.Args[1] {
	case "build":
		var err error
		client, err = app.NewBuildClient(static, templates, codeBlocks)
		if err != nil {
			log.Error("new build client", "error", err)
			os.Exit(1)
		}
	case "serve":
		var address string
		flag.StringVar(&address, "address", "127.0.0.1:8080", "address to listen on")
		flag.Parse()

		var err error
		client, err = app.NewServeClient(address, static, templates, codeBlocks)
		if err != nil {
			log.Error("new serve client", "error", err)
			os.Exit(1)
		}
	default:
		log.Error("unknown command, expected usage: millhouse <command>")
		os.Exit(1)
	}

	go func() {
		if err := client.Run(ctx, stop); err != nil {
			log.Error("app startup", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	stop()
	log.Info("stopping down gracefully, press Ctrl+C again to force")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := client.Stop(ctx, cancel); err != nil {
			log.Error("app shutdown", "error", err)
		}
	}()

	select {
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Error("timeout exceeded, forcing shutdown")
			os.Exit(1)
		}

		os.Exit(0)
	}
}
