package main

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"

	"github.com/go-faker/faker/v4"
)

type post struct {
	ID       int64  `db:"post_id" faker:"-"`
	Title    string `db:"title" faker:"sentence"`
	Content  string `db:"content" faker:"paragraph"`
	PubDate  string `db:"pub_date" faker:"timestamp"`
	AuthorID int64  `db:"author_id"`
}

var (
	posts []*post
)

func init() {
	posts = make([]*post, 0, 1000)
	for i := 0; i < 1000; i++ {
		p := &post{}
		if err := faker.FakeData(p); err != nil {
			panic(err)
		}
		p.AuthorID = rand.Int63n(100)
		posts = append(posts, p)
	}
}

func getPost() *post {
	return posts[rand.Intn(len(posts))]
}

func insertPost(ctx context.Context, db *DB, p *post) error {
	_, err := db.NamedExecContext(ctx, `
		INSERT INTO posts (title, content, pub_date, author_id)
		VALUES (:title, :content, :pub_date, :author_id)
	`, p)
	return err
}

func selectPost(ctx context.Context, db *DB, id int64) error {
	p := &post{}

	err := db.GetContext(ctx, p, `select * from posts where post_id = ?`, id)
	if err == sql.ErrNoRows || err == context.DeadlineExceeded {
		err = nil
	}
	return err
}

// 按照指定概率随机返回真假
//
// percent 1 ~ 100
// randomBool(80) 80%概率返回真
func randomBool(percent int) bool {
	if percent < 0 || percent > 100 {
		panic(errors.New("percent must be 1 ~ 100"))
	}

	switch percent {
	case 0:
		return false
	case 100:
		return true
	default:
		return rand.Intn(100) < percent
	}
}
