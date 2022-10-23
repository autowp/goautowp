package goautowp

import (
	"context"
	"database/sql"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

const ArticlesPreviewBaseUrl = "/img/articles/preview/"

type ArticlesGRPCServer struct {
	UnimplementedArticlesServer
	db *goqu.Database
}

func NewArticlesGRPCServer(db *goqu.Database) *ArticlesGRPCServer {
	return &ArticlesGRPCServer{
		db: db,
	}
}

func (s *ArticlesGRPCServer) GetList(ctx context.Context, in *ArticlesRequest) (*ArticlesResponse, error) {

	type row struct {
		Id              int64         `db:"id"`
		Name            string        `db:"name"`
		AuthorId        sql.NullInt64 `db:"author_id"`
		Catname         string        `db:"catname"`
		AddDate         time.Time     `db:"add_date"`
		PreviewFilename string        `db:"preview_filename"`
		Description     string        `db:"description"`
	}

	rows := make([]row, 0)

	paginator := util.Paginator{
		SQLSelect: s.db.Select("id", "name", "author_id", "catname", "add_date", "preview_filename", "description").From("articles").
			Where(goqu.L("enabled")).
			Order(goqu.I("add_date").Desc()),
	}

	sqlSelect, err := paginator.GetItemsByPage(ctx, int32(in.Page))
	if err != nil {
		return nil, err
	}

	err = sqlSelect.ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, err
	}

	articles := make([]*Article, 0)

	for _, article := range rows {

		authorId := article.AuthorId.Int64
		if !article.AuthorId.Valid {
			authorId = 0
		}

		articles = append(articles, &Article{
			Id:         article.Id,
			Name:       article.Name,
			AuthorId:   authorId,
			Catname:    article.Catname,
			Date:       timestamppb.New(article.AddDate),
			PreviewUrl: ArticlesPreviewBaseUrl + article.PreviewFilename,
		})
	}

	pages, err := paginator.GetPages(ctx)
	if err != nil {
		return nil, err
	}

	return &ArticlesResponse{
		Items: articles,
		Paginator: &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
		},
	}, nil
}

func (s *ArticlesGRPCServer) GetItemByCatname(ctx context.Context, in *ArticleByCatnameRequest) (*Article, error) {

	type row struct {
		Id              int64          `db:"id"`
		Name            string         `db:"name"`
		AuthorId        sql.NullInt64  `db:"author_id"`
		Catname         string         `db:"catname"`
		AddDate         time.Time      `db:"add_date"`
		PreviewFilename string         `db:"preview_filename"`
		Html            sql.NullString `db:"html"`
	}

	article := row{}

	success, err := s.db.Select(
		"articles.id", "articles.name", "articles.author_id", "articles.catname", "articles.add_date",
		"articles.preview_filename", "htmls.html",
	).
		From("articles").
		LeftJoin(
			goqu.T("htmls"),
			goqu.On(goqu.Ex{"articles.html_id": goqu.I("htmls.id")}),
		).
		Where(goqu.L("enabled"), goqu.I("catname").Eq(in.Catname)).
		ScanStructContext(ctx, &article)

	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint: nilnil
	}

	authorId := article.AuthorId.Int64
	if !article.AuthorId.Valid {
		authorId = 0
	}

	html := article.Html.String
	if !article.Html.Valid {
		html = ""
	}

	return &Article{
		Id:         article.Id,
		Name:       article.Name,
		AuthorId:   authorId,
		Catname:    article.Catname,
		Date:       timestamppb.New(article.AddDate),
		PreviewUrl: ArticlesPreviewBaseUrl + article.PreviewFilename,
		Html:       html,
	}, nil
}
