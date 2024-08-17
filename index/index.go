package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	topBrandsCount      = 150
	topPersonsCount     = 5
	topFactoriesCount   = 8
	topCategoriesCount  = 15
	topTwinsBrandsCount = 20
	brandsCacheKey      = "GO_TOPBRANDSLIST_3_%s"
	twinsCacheKey       = "GO_TWINS_5_%s"
	categoriesCacheKey  = "GO_CATEGORIES_6_%s"
	personsCacheKey     = "GO_PERSONS_3_%d_%s"
	factoriesCacheKey   = "GO_FACTORIES_3_%s"
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

	var cache BrandsCache

	options := items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly:            true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		TypeID:     []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		Limit:      topBrandsCount,
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

	return s.redis.Set(ctx, fmt.Sprintf(brandsCacheKey, lang), string(cacheBytes), 0).Err()
}

func (s *Index) BrandsCache(ctx context.Context, lang string) (BrandsCache, error) {
	var cache BrandsCache

	item, err := s.redis.Get(ctx, fmt.Sprintf(brandsCacheKey, lang)).Result()
	if err == nil {
		err = json.Unmarshal([]byte(item), &cache) //nolint: musttag
	} else if errors.Is(err, redis.Nil) {
		err = nil
	}

	return cache, err
}

func (s *Index) GenerateTwinsCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index twins cache for `%s`", lang)

	var (
		err       error
		twinsData TwinsCache
	)

	twinsData.Res, _, err = s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly: true,
		},
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ParentItemsListOptions{
				ParentItems: &items.ListOptions{
					TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDTwins},
					Fields: items.ListFields{
						ItemsCount:    true,
						NewItemsCount: true,
					},
				},
			},
		},
		TypeID:  []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		Limit:   topTwinsBrandsCount,
		OrderBy: []exp.OrderedExpression{goqu.C("items_count").Desc()},
	}, false)
	if err != nil {
		return err
	}

	twinsData.Count, err = s.repository.CountDistinct(ctx, items.ListOptions{
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ParentItemsListOptions{
				ParentItems: &items.ListOptions{
					TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDTwins},
				},
			},
		},
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
	})
	if err != nil {
		return err
	}

	cacheBytes, err := json.Marshal(twinsData) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, fmt.Sprintf(twinsCacheKey, lang), string(cacheBytes), 0).Err()
}

func (s *Index) TwinsCache(ctx context.Context, lang string) (TwinsCache, error) {
	twinsData := TwinsCache{}

	item, err := s.redis.Get(ctx, fmt.Sprintf(twinsCacheKey, lang)).Result()
	if err == nil {
		err = json.Unmarshal([]byte(item), &twinsData) //nolint: musttag
	} else if errors.Is(err, redis.Nil) {
		err = nil
	}

	return twinsData, err
}

func (s *Index) GenerateCategoriesCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index categories cache for `%s`", lang)

	var (
		err error
		res []items.Item
	)

	res, _, err = s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly:            true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		NoParents: true,
		TypeID:    []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDCategory},
		Limit:     topCategoriesCount,
		OrderBy:   []exp.OrderedExpression{goqu.C("descendants_count").Desc()},
	}, false)
	if err != nil {
		return err
	}

	b, err := json.Marshal(res) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, fmt.Sprintf(categoriesCacheKey, lang), string(b), 0).Err()
}

func (s *Index) CategoriesCache(ctx context.Context, lang string) ([]items.Item, error) {
	var res []items.Item

	item, err := s.redis.Get(ctx, fmt.Sprintf(categoriesCacheKey, lang)).Result()
	if err == nil {
		err = json.Unmarshal([]byte(item), &res) //nolint: musttag
	} else if errors.Is(err, redis.Nil) {
		err = nil
	}

	return res, err
}

func (s *Index) GeneratePersonsCache(
	ctx context.Context, pictureItemType pictures.ItemPictureType, lang string,
) error {
	logrus.Infof("generate index persons cache for `%s`", lang)

	var res []items.Item

	res, _, err := s.repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly: true,
		},
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDPerson},
		DescendantPictures: &items.ItemPicturesOptions{
			TypeID: pictureItemType,
			Pictures: &items.PicturesOptions{
				Status: pictures.StatusAccepted,
			},
		},
		Limit:   topPersonsCount,
		OrderBy: []exp.OrderedExpression{goqu.L("COUNT(1)").Desc()},
	}, false)
	if err != nil {
		return err
	}

	b, err := json.Marshal(res) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, fmt.Sprintf(personsCacheKey, pictureItemType, lang), string(b), 0).Err()
}

func (s *Index) PersonsCache(
	ctx context.Context, pictureItemType pictures.ItemPictureType, lang string,
) ([]items.Item, error) {
	var res []items.Item

	item, err := s.redis.Get(ctx, fmt.Sprintf(personsCacheKey, pictureItemType, lang)).Result()
	if err == nil {
		err = json.Unmarshal([]byte(item), &res) //nolint: musttag
	} else if errors.Is(err, redis.Nil) {
		err = nil
	}

	return res, err
}

func (s *Index) GenerateFactoriesCache(ctx context.Context, lang string) error {
	logrus.Infof("generate index factories cache for `%s`", lang)

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
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDFactory},
		ChildItems: &items.ParentItemsListOptions{
			ChildItems: &items.ListOptions{
				TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDEngine},
			},
		},
		Limit:   topFactoriesCount,
		OrderBy: []exp.OrderedExpression{goqu.L("COUNT(1)").Desc()},
	}, false)
	if err != nil {
		return err
	}

	b, err := json.Marshal(res) //nolint: musttag
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, fmt.Sprintf(factoriesCacheKey, lang), string(b), 0).Err()
}

func (s *Index) FactoriesCache(ctx context.Context, lang string) ([]items.Item, error) {
	var res []items.Item

	item, err := s.redis.Get(ctx, fmt.Sprintf(factoriesCacheKey, lang)).Result()
	if err == nil {
		err = json.Unmarshal([]byte(item), &res) //nolint: musttag
	} else if errors.Is(err, redis.Nil) {
		err = nil
	}

	return res, err
}
