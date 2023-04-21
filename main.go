package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"golang.org/x/exp/slog"
)

// WORKER 并发数
const WORKER = 4

func init() {
	slog.SetDefault(slog.New(slog.HandlerOptions{
		Level: slog.LevelDebug,
	}.NewTextHandler(os.Stderr)))
}

func main() {
	// writeTPS()
	// readTPS()
	readWriteTPS()
}

func writeTPS() {
	fmt.Println("WRITE TPS:")

	pragmas := []Pragma{
		{},
		{
			WithMutex: true,
		},
		{
			BusyTimeout: 3000,
		},
		{
			JournalMode: "WAL",
			Synchronous: "NORMAL",
		},
		{
			BusyTimeout: 3000,
			JournalMode: "WAL",
			Synchronous: "NORMAL",
		},
		{
			BusyTimeout: 3000,
			JournalMode: "WAL",
			Synchronous: "OFF",
		},
	}
	dirvers := []string{
		"sqlite",
		"sqlite3",
	}

	for _, pragma := range pragmas {
		for _, driver := range dirvers {
			path, db, err := newTestDB(driver, pragma)
			if err != nil {
				panic(fmt.Errorf("prepare test database, %w", err))
			}
			defer os.RemoveAll(path)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			tps := newTPS(ctx, WORKER, func(ctx context.Context) error {
				return insertArticle(ctx, db, getArticles())
			})

			fmt.Println("")
			fmt.Printf("%s:%s\n", db.DriverName(), db.dsn)
			fmt.Println(tps)
		}
	}
}

func readTPS() {
	pragmas := []Pragma{
		{},
		{
			BusyTimeout: 3000,
			JournalMode: "WAL",
			Synchronous: "NORMAL",
		},
		{
			BusyTimeout: 3000,
			JournalMode: "WAL",
			Synchronous: "NORMAL",
			TempStore:   "MEMORY",
			CacheSize:   10000,
		},
	}
	dirvers := []string{
		"sqlite",
		"sqlite3",
	}

	for _, pragma := range pragmas {
		for _, driver := range dirvers {
			path, db, err := newTestDB(driver, pragma)
			if err != nil {
				panic(fmt.Errorf("prepare test database, %w", err))
			}
			defer os.RemoveAll(path)

			for _, p := range articles {
				if err := insertArticle(context.Background(), db, p); err != nil {
					panic(fmt.Errorf("insert post, %w", err))
				}
			}
			if _, err := db.Exec("vacuum"); err != nil {
				panic(fmt.Errorf("vacuum, %w", err))
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			tps := newTPS(ctx, WORKER, func(ctx context.Context) error {
				return selectArticle(ctx, db, int64(rand.Intn(len(articles))))
			})

			fmt.Println("")
			fmt.Printf("%s:%s\n", db.DriverName(), db.dsn)
			fmt.Println(tps)
		}
	}
}

func readWriteTPS() {
	dirvers := []string{
		"sqlite",
		"sqlite3",
	}

	pragma := Pragma{
		// WithMutex:   true,
		BusyTimeout: 3000,
		JournalMode: "WAL",
		Synchronous: "NORMAL",
		TempStore:   "MEMORY",
		CacheSize:   10000,
	}

	worker := 4
	for _, percent := range []int{0, 10, 30, 50, 70, 90, 100} {
		for _, driver := range dirvers {
			path, db, err := newTestDB(driver, pragma)
			if err != nil {
				panic(fmt.Errorf("prepare test database, %w", err))
			}
			defer os.RemoveAll(path)

			for _, p := range articles {
				if err := insertArticle(context.Background(), db, p); err != nil {
					panic(fmt.Errorf("insert post, %w", err))
				}
			}
			if _, err := db.Exec("vacuum"); err != nil {
				panic(fmt.Errorf("vacuum, %w", err))
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			tps := newTPS(ctx, worker, func(ctx context.Context) error {
				if randomBool(percent) {
					return insertArticle(ctx, db, getArticles())
				}
				return selectArticle(ctx, db, int64(rand.Intn(len(articles))))
			})

			fmt.Println("")
			fmt.Printf("write percent: %d%%\n", percent)
			fmt.Printf("%s:%s\n", db.DriverName(), db.dsn)
			fmt.Println(tps)
		}
	}
}
