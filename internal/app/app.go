package app

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	log "log/slog"
	"net/http"
	"os"
	"text/template"
	"time"
)

type Client struct {
	server     *http.Server
	mux        *http.ServeMux
	static     fs.FS
	templates  fs.FS
	codeBlocks fs.FS
}

func NewClient(address string, static fs.FS, templates fs.FS, codeBlocks fs.FS) (*Client, error) {
	var handler http.Handler = http.NewServeMux()
	mux, ok := handler.(*http.ServeMux)
	if !ok {
		return nil, fmt.Errorf("handler is not a *http.ServeMux: %T", handler)
	}

	handler = NewRequestDuration(handler)
	handler = NewMethodCheck(handler)

	return &Client{
		server: &http.Server{
			Addr:              address,
			Handler:           handler,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       5 * time.Second,
		},
		mux:        mux,
		static:     static,
		templates:  templates,
		codeBlocks: codeBlocks,
	}, nil
}

func (c Client) Startup(ctx context.Context) error {
	c.registerRoutes()

	if err := c.server.ListenAndServe(); err != nil {
		log.Error("listen and serve", "error", err)
		os.Exit(1)
	}
	return nil
}

func (c Client) Shutdown(ctx context.Context, cancel context.CancelFunc) error {
	if err := c.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("app client shutdown: %w", err)
	}

	cancel()
	return nil
}

func (c Client) render(name string, w http.ResponseWriter, data any) error {
	tmpl, err := template.New(name).ParseFS(c.templates, "templates/layout.html", fmt.Sprintf("templates/%s.html", name))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// Render to a buffer first so we can see if there are any errors before writing to the response.
	resp := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&resp, "layout", data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// If there were no errors above, we can write the response.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := resp.WriteTo(w); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
