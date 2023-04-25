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
		{
			BusyTimeout: 0, // github.com/mattn/go-sqlite3 默认busy_timeout=5000，需要设置为0才公平
		},
		{
			MaxOpenConns: 1,
			BusyTimeout:  0,
		},
		{
			WithMutex:   true,
			BusyTimeout: 0,
		},
		{
			BusyTimeout: 3000,
		},
		{
			MaxOpenConns: 1,
			BusyTimeout:  0,
			JournalMode:  "WAL",
			Synchronous:  "NORMAL",
		},
		{
			MaxOpenConns: 2,
			BusyTimeout:  0,
			JournalMode:  "WAL",
			Synchronous:  "NORMAL",
		},
		{
			BusyTimeout: 0,
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
			if pragma.MaxOpenConns > 0 {
				fmt.Printf("MaxOpenConns: %d\n", pragma.MaxOpenConns)
			}
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

func sqliteTPS() {
	dirvers := []string{
		"sqlite",
		"sqlite3",
	}

	pragmas := []Pragma{
		// {
		// 	MaxOpenConns: 1,
		// 	BusyTimeout:  0,
		// 	JournalMode:  "WAL",
		// 	Synchronous:  "NORMAL",
		// 	TempStore:    "MEMORY",
		// 	CacheSize:    10000,
		// },
		{
			BusyTimeout: 3000,
			JournalMode: "WAL",
			Synchronous: "NORMAL",
			TempStore:   "MEMORY",
			CacheSize:   10000,
		},
	}

	for _, percent := range []int{0, 30, 50, 70, 100} {
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

				fmt.Println("")
				fmt.Printf("write percent: %d%%\n", percent)
				if n := pragma.MaxOpenConns; n > 0 {
					fmt.Printf("MaxOpenConns: %d\n", n)
				}
				fmt.Printf("%s:%s\n", db.DriverName(), db.dsn)

				tps := newTPS(ctx, WORKER, func(ctx context.Context) error {
					if randomBool(percent) {
						return insertArticle(ctx, db, getArticle())
					}
					return selectArticle(ctx, db, int64(rand.Intn(len(articles))))
				})

				fmt.Println(tps)
			}
		}
	}
}
