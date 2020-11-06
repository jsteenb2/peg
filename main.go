package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var (
	version = ""
	commit  = ""
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		cancel()
	}()

	rootCmd := cmd(ctx)
	rootCmd.ExecuteContext(ctx)
}
