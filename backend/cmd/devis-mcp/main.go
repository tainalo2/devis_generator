package main

import (
	"log/slog"
	"os"

	"github.com/tainalo2/devis_generator/backend/internal/mcpserver"
)

func main() {
	s := mcpserver.New()
	if err := s.Run(); err != nil {
		slog.Error("mcp", "error", err)
		os.Exit(1)
	}
}
