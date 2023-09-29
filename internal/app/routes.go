package app

import (
	"io"
	log "log/slog"
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/quick"
)

func (c *Client) registerRoutes() {
	c.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		file, err := c.static.Open("static/images/favicon.png")
		if err != nil {
			log.Error("open favicon", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "image/png")
		if _, err := io.Copy(w, file); err != nil {
			log.Error("copy favicon", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	c.mux.Handle("/static/", http.FileServer(http.FS(c.static)))

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
