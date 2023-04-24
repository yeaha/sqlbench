package main

import (
	"context"
	"errors"
	"math/rand"

	"github.com/go-faker/faker/v4"
)

type article struct {
	ID       int64  `db:"article_id" faker:"-"`
	Title    string `db:"title" faker:"sentence"`
	Content  string `db:"content" faker:"paragraph"`
	PubDate  string `db:"pub_date" faker:"timestamp"`
	AuthorID int64  `db:"author_id"`
}

var (
	articles []*article
)

func init() {
	articles = make([]*article, 0, 1000)
	for i := 0; i < 1000; i++ {
		a := &article{}
		if err := faker.FakeData(a); err != nil {
			panic(err)
		}
		a.AuthorID = rand.Int63n(100)
		articles = append(articles, a)
	}
}

func getArticle() *article {
	return articles[rand.Intn(len(articles))]
}

func insertArticle(ctx context.Context, db *DB, a *article) error {
	_, err := db.NamedExecContext(ctx, `
		INSERT INTO articles (title, content, pub_date, author_id)
		VALUES (:title, :content, :pub_date, :author_id)
	`, a)
	return err
}

func selectArticle(ctx context.Context, db *DB, id int64) error {
	p := &article{}

	return db.GetContext(ctx, p, `select * from articles where article_id = ?`, id)
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
