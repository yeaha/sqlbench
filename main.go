package main

import (
	"os"
	"time"

	"golang.org/x/exp/slog"
)

// WORKER 并发数
const WORKER = 4

var benchTime = 10 * time.Second

func init() {
	slog.SetDefault(slog.New(slog.HandlerOptions{
		Level: slog.LevelDebug,
	}.NewTextHandler(os.Stderr)))
}

func main() {
	// sqliteReadTPS()
	// sqliteWriteTPS()
	// sqliteReadWriteTPS()

	pgsqlReadTPS()
	// pgsqlWriteTPS()
}
