package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
)

func sqliteWriteTPS() {
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

			ctx, cancel := context.WithTimeout(context.Background(), benchTime)
			defer cancel()

			tps := newTPS(ctx, WORKER, func(ctx context.Context) error {
				return insertArticle(ctx, db, getArticle())
			})

			fmt.Println("")
			fmt.Printf("%s:%s\n", db.DriverName(), db.dsn)
			fmt.Println(tps)
		}
	}
}

func sqliteReadTPS() {
	fmt.Println("READ TPS:")

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

			ctx, cancel := context.WithTimeout(context.Background(), benchTime)
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

func sqliteReadWriteTPS() {
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

			ctx, cancel := context.WithTimeout(context.Background(), benchTime)
			defer cancel()

			tps := newTPS(ctx, WORKER, func(ctx context.Context) error {
				if randomBool(percent) {
					return insertArticle(ctx, db, getArticle())
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
