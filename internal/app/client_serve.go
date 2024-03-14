package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	log "log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/chroma/quick"
)

type ServeClient struct {
	server     *http.Server
	mux        *http.ServeMux
	static     fs.FS
	templates  fs.FS
	codeBlocks fs.FS
}

func NewServeClient(address string, static fs.FS, templates fs.FS, codeBlocks fs.FS) (*ServeClient, error) {
	var handler http.Handler = http.NewServeMux()
	mux, ok := handler.(*http.ServeMux)
	if !ok {
		return nil, fmt.Errorf("handler is not a *http.ServeMux: %T", handler)
	}

	handler = NewRequestDuration(handler)
	handler = NewMethodCheck(handler)

	return &ServeClient{
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

func (c ServeClient) Run(ctx context.Context, cancel context.CancelFunc) error {
	c.registerRoutes()

	log.Info(fmt.Sprintf("listening on http://%s", c.server.Addr))

	if err := c.server.ListenAndServe(); err != nil {
		log.Error("listen and serve", "error", err)
	}
	return nil
}

func (c ServeClient) Stop(ctx context.Context, cancel context.CancelFunc) error {
	if err := c.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("client stop: %w", err)
	}

	cancel()
	return nil
}

func (c ServeClient) render(name string, w http.ResponseWriter, data any) error {
	tmpl, err := template.New(name).ParseFS(c.templates, "layout.html", fmt.Sprintf("%s.html", name))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// Render to a buffer first so we can see if there are any errors before writing to the response.
	resp := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&resp, "layout", data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// If there were no errors above, we can write the output.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := resp.WriteTo(w); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func (c *ServeClient) registerRoutes() {
	c.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		file, err := c.static.Open("root/favicon.ico")
		if err != nil {
			log.Error("open favicon", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "image/x-icon")
		if _, err := io.Copy(w, file); err != nil {
			log.Error("copy favicon", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	c.mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		file, err := c.static.Open("root/robots.txt")
		if err != nil {
			log.Error("open robots.txt", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "text/plain")
		if _, err := io.Copy(w, file); err != nil {
			log.Error("copy robots.txt", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	c.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(c.static))))
	// Use this if embed.FS is being used, also need to add the prefixes to all the files if embed.FS is used.
	// c.mux.Handle("/static/", http.FileServer(http.FS(c.static)))

	c.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/not-found", http.StatusSeeOther)
			return
		}

		if err := c.render("index", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("contact", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("error", w, nil); err != nil {
			log.Error("render", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	c.mux.HandleFunc("/licenses", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("licenses", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/meetups", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("meetups", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/not-found", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("not-found", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/open-source", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("open-source", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("posts", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	// Redirect old post URL to new post URL.
	c.mux.HandleFunc("/posts/graceful-shutdowns-in-golang-with-signal-notify-context", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts/graceful-shutdowns-with-signal-notify-context", http.StatusSeeOther)
	})
	c.mux.HandleFunc("/posts/graceful-shutdowns-with-signal-notify-context", func(w http.ResponseWriter, r *http.Request) {
		file, err := c.codeBlocks.Open("code-blocks/graceful-shutdowns-with-signal-notify-context/main.go")
		if err != nil {
			log.Error("open favicon", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			log.Error("read file", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		example := &strings.Builder{}
		if err := quick.Highlight(example, string(data), "go", "html", "base16-snazzy"); err != nil {
			log.Error("highlight", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		content := struct {
			Example string
		}{
			Example: example.String(),
		}

		if err := c.render("posts/graceful-shutdowns-with-signal-notify-context", w, content); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})

	c.mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		if err := c.render("projects", w, nil); err != nil {
			log.Error("render", "error", err)
			http.Redirect(w, r, "/error", http.StatusSeeOther)
			return
		}
	})
}
