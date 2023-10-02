package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/alecthomas/chroma/quick"
)

type BuildClient struct {
	static     fs.FS
	templates  fs.FS
	codeBlocks fs.FS
}

func NewBuildClient(static fs.FS, templates fs.FS, codeBlocks fs.FS) (*BuildClient, error) {
	return &BuildClient{
		static:     static,
		templates:  templates,
		codeBlocks: codeBlocks,
	}, nil
}

func (c BuildClient) Run(ctx context.Context, cancel context.CancelFunc) error {
	defer cancel()
	outDir := "./dist"

	if err := c.initOutDir(outDir); err != nil {
		return fmt.Errorf("client run: %w", err)
	}

	if err := c.copyAll("static", outDir); err != nil {
		return fmt.Errorf("client run: %w", err)
	}

	if err := c.renderAll("templates", outDir); err != nil {
		return fmt.Errorf("client run: %w", err)
	}

	return nil
}

func (c BuildClient) Stop(ctx context.Context, cancel context.CancelFunc) error {
	cancel()
	return nil
}

func (c BuildClient) initOutDir(outDir string) error {
	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("client run: %w", err)
	}
	if err := os.Mkdir(outDir, 0o777); err != nil {
		return fmt.Errorf("client run: %w", err)
	}
	return nil
}

func (c BuildClient) copyAll(src, dst string) error {
	if err := fs.WalkDir(c.static, src, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir: %w", err)
		}

		dstPath := fmt.Sprintf("%s/%s", dst, srcPath)

		if d.IsDir() {
			if err := os.Mkdir(dstPath, 0o777); err != nil {
				return fmt.Errorf("walk dir: %w", err)
			}
		} else {
			if err := c.copy(srcPath, dstPath); err != nil {
				return fmt.Errorf("walk dir: %w", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("copy all: %w", err)
	}

	return nil
}

func (c BuildClient) copy(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return d.Sync()
}

func (c BuildClient) renderAll(src, dst string) error {
	if err := fs.WalkDir(c.templates, src, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir: %w", err)
		}
		if srcPath == "templates/layout.html" {
			return nil
		}

		relPath := strings.TrimPrefix(srcPath, src)
		dstPath := dst
		if relPath != "" {
			dstPath = fmt.Sprintf("%s%s", dst, relPath)
		}

		if d.IsDir() {
			if err := os.Mkdir(dstPath, 0o777); err != nil && !errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("walk dir: %w", err)
			}
		} else {
			// TODO: Fix, this is gross.
			var content any
			if srcPath == "templates/posts/graceful-shutdowns-with-signal-notify-context.html" {
				file, err := c.codeBlocks.Open("code-blocks/graceful-shutdowns-with-signal-notify-context/main.go")
				if err != nil {
					return fmt.Errorf("walk dir: %w", err)
				}
				defer file.Close()

				data, err := io.ReadAll(file)
				if err != nil {
					return fmt.Errorf("walk dir: %w", err)
				}

				example := &strings.Builder{}
				if err := quick.Highlight(example, string(data), "go", "html", "base16-snazzy"); err != nil {
					return fmt.Errorf("walk dir: %w", err)
				}

				content = struct {
					Example string
				}{
					Example: example.String(),
				}
			}
			if err := c.render(srcPath, dstPath, content); err != nil {
				return fmt.Errorf("walk dir: %w", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("render all: %w", err)
	}

	return nil
}

func (c BuildClient) render(src string, dst string, data any) error {
	tmpl, err := template.New(src).ParseFS(c.templates, "templates/layout.html", src)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// Render to a buffer first so we can see if there are any errors before writing to the response.
	resp := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&resp, "layout", data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	d, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	defer d.Close()

	// If there were no errors above, we can write the output.
	if _, err := resp.WriteTo(d); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
