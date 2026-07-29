package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func serveStatic(ctx context.Context, dir, addr string) error {
	root, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve static site directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("inspect static site directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("static site path is not a directory: %s", root)
	}

	server := &http.Server{Addr: addr, Handler: staticHandler(root)}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown static server: %w", err)
		}
		err := <-serverErr
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func staticHandler(root string) http.Handler {
	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
}
