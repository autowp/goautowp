package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	TopBrandsCount      = 150
	TopPersonsCount     = 5
	TopFactoriesCount   = 8
	TopCategoriesCount  = 15
	TopTwinsBrandsCount = 20
)

type Index struct {
	redis      *redis.Client
	repository *items.Repository
}

type BrandsCache struct {
	Items []items.Item
	Total int
}

type TwinsCache struct {
	Count int
	Res   []items.Item
}

func NewIndex(redis *redis.Client, repository *items.Repository) *Index {
	return &Index{
		redis:      redis,
		repository: repository,
	}
}

func (s *Index) GenerateBrandsCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index brands cache for `%s`", lang)

	key := "GO_TOPBRANDSLIST_3_" + lang

	var cache BrandsCache

	options := items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly:            true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		TypeID:     []items.ItemType{items.BRAND},
		Limit:      TopBrandsCount,
		OrderBy:    []exp.OrderedExpression{goqu.C("descendants_count").Desc()},
		SortByName: true,
	}

	list, _, err := s.repository.List(ctx, options, false)
	if err != nil {
		return err
	}

	count, err := s.repository.Count(ctx, options)
	if err != nil {
		return err
	}

	cache.Items = list
	cache.Total = count

	cacheBytes, err := json.Marshal(cache) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, string(cacheBytes), 0).Err()
}

func (s *Index) BrandsCache(ctx context.Context, lang string) (BrandsCache, error) {
	key := "GO_TOPBRANDSLIST_3_" + lang

	var cache BrandsCache

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return cache, err
	}

	if err == nil {
		err = json.Unmarshal([]byte(item), &cache) //nolint: musttag
		if err != nil {
			return cache, err
		}
	}

	return cache, nil
}

func (s *Index) GenerateTwinsCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index twins cache for `%s`", lang)

	var err error

	key := "GO_TWINS_5_" + lang

	twinsData := struct {
		Count int
		Res   []items.Item
	}{
		0,
		nil,
	}

	twinsData.Res, _, err = s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly: true,
		},
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ListOptions{
				TypeID: []items.ItemType{items.TWINS},
				Fields: items.ListFields{
					ItemsCount:    true,
					NewItemsCount: true,
				},
			},
		},
		TypeID:  []items.ItemType{items.BRAND},
		Limit:   TopTwinsBrandsCount,
		OrderBy: []exp.OrderedExpression{goqu.C("items_count").Desc()},
	}, false)
	if err != nil {
		return err
	}

	twinsData.Count, err = s.repository.CountDistinct(ctx, items.ListOptions{
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ListOptions{
				TypeID: []items.ItemType{items.TWINS},
			},
		},
		TypeID: []items.ItemType{items.BRAND},
	})
	if err != nil {
		return err
	}

	cacheBytes, err := json.Marshal(twinsData) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, string(cacheBytes), 0).Err()
}

func (s *Index) TwinsCache(ctx context.Context, lang string) (TwinsCache, error) {
	key := "GO_TWINS_5_" + lang
	twinsData := TwinsCache{}

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return twinsData, err
	}

	if err == nil {
		err = json.Unmarshal([]byte(item), &twinsData) //nolint: musttag
	}

	return twinsData, err
}

func (s *Index) GenerateCategoriesCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index categories cache for `%s`", lang)

	var err error

	key := "GO_CATEGORIES_6_" + lang

	var res []items.Item

	res, _, err = s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly:            true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		NoParents: true,
		TypeID:    []items.ItemType{items.CATEGORY},
		Limit:     TopCategoriesCount,
		OrderBy:   []exp.OrderedExpression{goqu.C("descendants_count").Desc()},
	}, false)
	if err != nil {
		return err
	}

	b, err := json.Marshal(res) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, string(b), 0).Err()
}

func (s *Index) CategoriesCache(ctx context.Context, lang string) ([]items.Item, error) {
	key := "GO_CATEGORIES_6_" + lang

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	var res []items.Item

	if err == nil {
		err = json.Unmarshal([]byte(item), &res) //nolint: musttag
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (s *Index) GeneratePersonsCache(
	ctx context.Context, pictureItemType pictures.ItemPictureType, lang string,
) error {
	logrus.Infof("generate index persons cache for `%s`", lang)

	key := fmt.Sprintf("GO_PERSONS_3_%d_%s", pictureItemType, lang)

	var res []items.Item

	res, _, err := s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly: true,
		},
		TypeID: []items.ItemType{items.PERSON},
		DescendantPictures: &items.ItemPicturesOptions{
			TypeID: pictureItemType,
			Pictures: &items.PicturesOptions{
				Status: pictures.StatusAccepted,
			},
		},
		Limit:   TopPersonsCount,
		OrderBy: []exp.OrderedExpression{goqu.L("COUNT(1)").Desc()},
	}, false)
	if err != nil {
		return err
	}

	b, err := json.Marshal(res) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, string(b), 0).Err()
}

func (s *Index) PersonsCache(
	ctx context.Context, pictureItemType pictures.ItemPictureType, lang string,
) ([]items.Item, error) {
	key := fmt.Sprintf("GO_PERSONS_3_%d_%s", pictureItemType, lang)

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	var res []items.Item

	if err == nil {
		err = json.Unmarshal([]byte(item), &res) //nolint: musttag
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (s *Index) GenerateFactoriesCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index factories cache for `%s`", lang)

	key := "GO_FACTORIES_3_" + lang

	var (
		res []items.Item
		err error
	)

	res, _, err = s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly:           true,
			ChildItemsCount:    true,
			NewChildItemsCount: true,
		},
		TypeID: []items.ItemType{items.FACTORY},
		ChildItems: &items.ListOptions{
			TypeID: []items.ItemType{items.VEHICLE, items.ENGINE},
		},
		Limit:   TopFactoriesCount,
		OrderBy: []exp.OrderedExpression{goqu.L("COUNT(1)").Desc()},
	}, false)
	if err != nil {
		return err
	}

	b, err := json.Marshal(res) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, string(b), 0).Err()
}

func (s *Index) FactoriesCache(ctx context.Context, lang string) ([]items.Item, error) {
	key := "GO_FACTORIES_3_" + lang

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	var res []items.Item

	if err == nil {
		err = json.Unmarshal([]byte(item), &res) //nolint: musttag
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}
