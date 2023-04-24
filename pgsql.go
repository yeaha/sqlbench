package main

import (
	"context"
	"fmt"
	"math/rand"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

func newPGSQLDB() (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", "postgres://bench@localhost:5432/bench?sslmode=disable")
	if err != nil {
		return nil, err
	}

	ddm := []string{
		`create table if not exists public.articles(
			article_id serial primary key,
			title text,
			content text,
			pub_date timestamp with time zone,
			author_id integer
		)`,
		`create index if not exists articles_author_id_idx on public.articles(author_id)`,
	}

	for _, v := range ddm {
		if _, err := db.ExecContext(context.Background(), v); err != nil {
			return nil, err
		}
	}

	return db.Unsafe(), nil
}

func pgsqlTPS() {
	db, err := newPGSQLDB()
	if err != nil {
		panic(fmt.Errorf("connect postgresql, %w", err))
	}
	defer db.Close()

	for _, v := range articles {
		_, err = db.NamedExecContext(context.Background(), `
			INSERT INTO public.articles (title, content, pub_date, author_id)
			VALUES (:title, :content, :pub_date, :author_id)
			`, v)
		if err != nil {
			panic(fmt.Errorf("prepare data, %w", err))
		}
	}

	for _, writePercent := range []int{0, 30, 50, 70, 100} {
		fmt.Println("")
		fmt.Printf("postgrsql write percent: %d%%\n", writePercent)

		ctx, cancel := context.WithTimeout(context.Background(), benchTime)
		defer cancel()

		tps := newTPS(ctx, WORKER, func(ctx context.Context) (err error) {
			if randomBool(writePercent) {
				_, err = db.NamedExecContext(ctx, `
					INSERT INTO public.articles (title, content, pub_date, author_id)
					VALUES (:title, :content, :pub_date, :author_id)
					`, getArticle())
			} else {
				p := &article{}
				err = db.GetContext(ctx, p, `select * from public.articles where article_id = $1`, rand.Int63n(int64(len(articles))))
			}
			return
		})

		fmt.Println(tps)
	}
}
