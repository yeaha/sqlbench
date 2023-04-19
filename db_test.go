package main

import (
	"context"
	"os"
	"testing"
)

func BenchmarkWriter(b *testing.B) {
	cases := []struct {
		Name   string
		Pragma Pragma
	}{
		{
			Name:   "default",
			Pragma: Pragma{},
		},
		{
			Name: "withoutMutex",
			Pragma: Pragma{
				BusyTimeout: 5000,
			},
		},
		{
			Name: "withoutMutex&wal",
			Pragma: Pragma{
				BusyTimeout: 5000,
				JournalMode: "WAL",
			},
		},
		{
			Name: "withoutMutex&wal&more",
			Pragma: Pragma{
				BusyTimeout: 5000,
				JournalMode: "WAL",
				Synchronous: "NORMAL",
				TempStore:   "MEMORY",
				MmapSize:    30000000000,
				CacheSize:   10000,
			},
		},
		{
			Name: "withMutex",
			Pragma: Pragma{
				WithMutex: true,
			},
		},
		{
			Name: "withMutex&wal",
			Pragma: Pragma{
				WithMutex:   true,
				JournalMode: "WAL",
			},
		},
		{
			Name: "withMutex&wal&more",
			Pragma: Pragma{
				WithMutex:   true,
				JournalMode: "WAL",
				Synchronous: "NORMAL",
				TempStore:   "MEMORY",
				MmapSize:    30000000000,
				CacheSize:   10000,
			},
		},
	}

	for _, driver := range []string{"sqlite", "sqlite3"} {
		b.Run(driver, func(b *testing.B) {
			for _, v := range cases {
				b.Run(v.Name, func(b *testing.B) {
					path, db, err := newTestDB(driver, v.Pragma)
					if err != nil {
						b.Fatalf("prepare database, %v", err)
					}
					// b.Logf("database path: %s", path)

					defer func() {
						os.RemoveAll(path)
					}()

					b.ResetTimer()
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							if err := insertPost(context.Background(), db, getPost()); err != nil {
								b.Fatalf("insert post, %v", err)
							}
						}
					})
				})
			}
		})
	}
}
