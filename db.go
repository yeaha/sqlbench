package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

// Pragma sqlite数据库配置
//
// https://www.sqlite.org/pragma.html
type Pragma struct {
	WithMutex bool

	BusyTimeout       int
	Cache             string
	CacheSize         int
	FullSync          bool
	JournalMode       string
	MmapSize          int
	Synchronous       string
	TempStore         string
	WALAutoCheckpoint int
}

func (p Pragma) encode(driver string) string {
	switch driver {
	case "sqlite3":
		return p.encodeMattn()
	case "sqlite":
		return p.encodeModernc()
	}
	return ""
}

func (p Pragma) encodeMattn() string {
	val := url.Values{}

	if v := p.JournalMode; v != "" {
		val.Set("_journal_mode", v)
	}
	if v := p.Synchronous; v != "" {
		val.Set("_synchronous", v)
	}
	if v := p.CacheSize; v != 0 {
		val.Set("_cache_size", fmt.Sprintf("%d", v))
	}
	if v := p.BusyTimeout; v != 0 {
		val.Set("_busy_timeout", fmt.Sprintf("%d", v))
	}
	if v := p.FullSync; v {
		val.Set("_fullsync", "1")
	}
	if v := p.TempStore; v != "" {
		val.Set("_temp_store", v)
	}
	if v := p.MmapSize; v != 0 {
		val.Set("_mmap_size", fmt.Sprintf("%d", v))
	}
	if v := p.Cache; v != "" {
		val.Set("cache", v)
	}
	if v := p.WALAutoCheckpoint; v != 0 {
		val.Set("_wal_autocheckpoint", fmt.Sprintf("%d", v))
	}

	result, _ := url.QueryUnescape(val.Encode())
	return result
}

func (p Pragma) encodeModernc() string {
	val := url.Values{}

	if v := p.JournalMode; v != "" {
		val.Add("_pragma", fmt.Sprintf("journal_mode(%s)", v))
	}
	if v := p.Synchronous; v != "" {
		val.Add("_pragma", fmt.Sprintf("synchronous(%s)", v))
	}
	if v := p.CacheSize; v != 0 {
		val.Add("_pragma", fmt.Sprintf("cache_size(%d)", v))
	}
	if v := p.BusyTimeout; v != 0 {
		val.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", v))
	}
	if v := p.FullSync; v {
		val.Add("_pragma", "fullsync(1)")
	}
	if v := p.TempStore; v != "" {
		val.Add("_pragma", fmt.Sprintf("temp_store(%s)", v))
	}
	if v := p.MmapSize; v != 0 {
		val.Add("_pragma", fmt.Sprintf("mmap_size(%d)", v))
	}
	if v := p.Cache; v != "" {
		val.Set("cache", v)
	}
	if v := p.WALAutoCheckpoint; v != 0 {
		val.Add("_pragma", fmt.Sprintf("wal_autocheckpoint(%d)", v))
	}

	result, _ := url.QueryUnescape(val.Encode())
	return result
}

// DB 数据库连接
type DB struct {
	*sync.RWMutex
	*sqlx.DB

	withMutex bool
	dsn       string
}

// NewDB 创建数据库连接
//
//	dirver=sqlite3 use github.com/mattn/go-sqlite3
//	driver=sqlite use modernc.org/sqlite
func NewDB(driver, file string, pragma Pragma) (*DB, error) {
	dsn := fmt.Sprintf("%s?%s", file, pragma.encode(driver))

	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		return nil, err
	}
	// slog.Debug("connect database", slog.String("dsn", dsn), slog.String("driver", driver))

	return &DB{
		RWMutex:   &sync.RWMutex{},
		DB:        db.Unsafe(),
		withMutex: pragma.WithMutex,
		dsn:       dsn,
	}, nil
}

// QueryContext 查询
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	if db.withMutex {
		db.RLock()
		defer db.RUnlock()
	}

	return db.DB.QueryxContext(ctx, query, args...)
}

// GetContext 查询单条
func (db *DB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	if db.withMutex {
		db.RLock()
		defer db.RUnlock()
	}

	return db.DB.GetContext(ctx, dest, query, args...)
}

// SelectContext 查询多条
func (db *DB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	if db.withMutex {
		db.RLock()
		defer db.RUnlock()
	}

	return db.DB.SelectContext(ctx, dest, query, args...)
}

// ExecContext 执行
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if db.withMutex {
		db.Lock()
		defer db.Unlock()
	}

	return db.DB.ExecContext(ctx, query, args...)
}

// NamedExecContext 执行
func (db *DB) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	if db.withMutex {
		db.Lock()
		defer db.Unlock()
	}

	return db.DB.NamedExecContext(ctx, query, arg)
}

// NamedQueryContext 查询
func (db *DB) NamedQueryContext(ctx context.Context, query string, arg any) (*sqlx.Rows, error) {
	if db.withMutex {
		db.RLock()
		defer db.RUnlock()
	}

	return db.DB.NamedQueryContext(ctx, query, arg)
}

func newTestDB(dirver string, pragme Pragma) (path string, db *DB, err error) {
	path, err = os.MkdirTemp("", "sqlite-*")
	if err != nil {
		err = fmt.Errorf("make temp dir, %w", err)
		return
	}

	defer func() {
		if err != nil {
			if removeErr := os.RemoveAll(path); removeErr != nil {
				err = errors.Join(err, removeErr)
			}
		}
	}()

	db, err = NewDB(dirver, filepath.Join(path, "test.db"), pragme)
	if err != nil {
		err = fmt.Errorf("connect database, %w", err)
		return
	}

	if err = prepareTestDB(db); err != nil {
		err = fmt.Errorf("prepare database, %w", err)
		return
	}
	return
}

func prepareTestDB(db *DB) error {
	ddm := []string{
		`CREATE TABLE IF NOT EXISTS articles (
			article_id INTEGER PRIMARY KEY,
			title TEXT,
			content TEXT,
			pub_date TEXT,
			author_id INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_articles_author_id ON articles (author_id)`,
	}

	for _, cmd := range ddm {
		if _, err := db.ExecContext(context.Background(), cmd); err != nil {
			return err
		}
	}
	return nil
}

// TPS 每秒事务完成度
type TPS struct {
	Worker   int
	Duration time.Duration
	Success  *atomic.Int64
	Error    *atomic.Int64
}

func (t *TPS) String() string {
	return fmt.Sprintf("duration: %s, worker: %d, success: %d, error: %d, tps: %.2f",
		t.Duration, t.Worker, t.Success.Load(), t.Error.Load(), float64(t.Success.Load())/t.Duration.Seconds())
}

func newTPS(ctx context.Context, worker int, fn func(context.Context) error) *TPS {
	startTime := time.Now()
	result := &TPS{
		Worker:  worker,
		Success: &atomic.Int64{},
		Error:   &atomic.Int64{},
	}

	var wg sync.WaitGroup
	for i := 0; i < worker; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := fn(ctx)
					if err == sql.ErrNoRows {
						err = nil
					}

					if err == nil {
						result.Success.Add(1)
					} else {
						result.Error.Add(1)
					}
				}
			}
		}()
	}
	wg.Wait()

	result.Duration = time.Since(startTime)
	return result
}
