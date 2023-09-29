package main

import (
	"context"
	"embed"
	"fmt"
	log "log/slog"
	"os"
	"os/signal"
	"time"

	"gitlab.com/strongishllama/millhouse.dev/internal/app"
)

//go:embed static
var static embed.FS

//go:embed templates
var templates embed.FS

//go:embed code-blocks
var codeBlocks embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.SetDefault(log.New(log.NewJSONHandler(os.Stdout, &log.HandlerOptions{
		Level: log.LevelDebug,
	})))

	// todo
	address := "127.0.0.1:8080"

	// TODO: Switch to embed.FS.
	tmpl := os.DirFS(".")

	client, err := app.NewClient(address, static, tmpl, codeBlocks)
	if err != nil {
		log.Error("new app client", "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info(fmt.Sprintf("listening on http://%s", address))
		if err := client.Startup(ctx); err != nil {
			log.Error("app startup", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	stop()
	log.Info("shutting down gracefully, press Ctrl+C again to force")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := client.Shutdown(ctx, cancel); err != nil {
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
