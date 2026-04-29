package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aleksclark/spec-torture/internal/mockserver"
)

func main() {
	s, err := mockserver.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start mock server: %v\n", err)
		os.Exit(1)
	}

	info := map[string]any{
		"url":  s.URL(),
		"port": s.Port(),
	}
	json.NewEncoder(os.Stdout).Encode(info)

	// Wait for signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	s.Close()
}
