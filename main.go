package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "mini-s3/store"
)

func main() {
    s := store.NewFileStore("./data")

    mux := http.NewServeMux()
    mux.HandleFunc("/", s.Handler)

    srv := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  0, // No timeout for large uploads
        WriteTimeout: 0, // No timeout for large downloads
    }

    go func() {
        log.Printf("mini-s3 listening on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s", err)
        }
    }()

    // graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    <-stop

    log.Println("shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server Shutdown Failed: %+v", err)
    }
    log.Println("server exited")
}
