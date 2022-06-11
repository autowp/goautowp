package util

import (
	"github.com/doug-martin/goqu/v9"
	"math"
)

const DefaultItemCountPerPage = 10

type Paginator struct {
	SQLSelect           *goqu.SelectDataset
	pageCount           int32
	pageCountCalculated bool
	ItemCountPerPage    int32
	CurrentPageNumber   int32
	itemCount           int32
	itemCountCalculated bool
}

type Pages struct {
	PageCount        int32
	ItemCountPerPage int32
	First            int32
	Current          int32
	Last             int32
	Previous         *int32
	Next             *int32
	FirstPageInRange int32
	LastPageInRange  int32
	TotalItemCount   int32
	PagesInRange     []int32
}

func (s *Paginator) Count() (int32, error) {
	var err error
	if !s.pageCountCalculated {
		s.pageCount, err = s.calculatePageCount()
		if err != nil {
			return 0, err
		}

		s.pageCountCalculated = true
	}

	return s.pageCount, nil
}

func (s *Paginator) calculatePageCount() (int32, error) {
	count, err := s.GetTotalItemCount()
	if err != nil {
		return 0, err
	}

	return int32(math.Ceil(float64(count) / float64(s.getItemCountPerPage()))), nil
}

func (s *Paginator) calculateCount() (int32, error) {
	res, err := s.SQLSelect.ClearOrder().
		ClearOffset().
		ClearLimit().
		ClearSelect().
		Prepared(true).
		Count()

	if err != nil {
		return 0, err
	}

	return int32(res), nil
}

func (s *Paginator) getItemCountPerPage() int32 {
	if s.ItemCountPerPage <= 0 {
		s.ItemCountPerPage = DefaultItemCountPerPage
	}

	return s.ItemCountPerPage
}

func MinMax(array []int32) (int32, int32) {
	var max, min = array[0], array[0]

	for _, value := range array {
		if max < value {
			max = value
		}

		if min > value {
			min = value
		}
	}

	return min, max
}

func (s *Paginator) GetPages() (*Pages, error) {
	pageCount, err := s.Count()

	if err != nil {
		return nil, err
	}

	currentPageNumber, err := s.getCurrentPageNumber()

	if err != nil {
		return nil, err
	}

	totalItemCount, err := s.GetTotalItemCount()
	if err != nil {
		return nil, err
	}

	pages := Pages{
		PageCount:        pageCount,
		ItemCountPerPage: s.getItemCountPerPage(),
		First:            1,
		Current:          currentPageNumber,
		Last:             pageCount,
		Previous:         nil,
		Next:             nil,
		TotalItemCount:   totalItemCount,
	}

	// Previous and next
	if currentPageNumber-1 > 0 {
		previous := currentPageNumber - 1
		pages.Previous = &previous
	}

	if currentPageNumber+1 <= pageCount {
		next := currentPageNumber + 1
		pages.Next = &next
	}

	// Pages in range
	var pageRange int32 = 10

	pageNumber := currentPageNumber

	if pageRange > pageCount {
		pageRange = pageCount
	}

	delta := int32(math.Ceil(float64(pageRange) / 2.0))

	lowerBound := pageCount - pageRange + 1
	upperBound := pageCount

	if pageNumber-delta <= pageCount-pageRange {
		if pageNumber-delta < 0 {
			delta = pageNumber
		}

		offset := pageNumber - delta
		lowerBound = offset + 1
		upperBound = offset + pageRange
	}

	pagesInRange, err := s.getPagesInRange(lowerBound, upperBound)
	if err != nil {
		return nil, err
	}

	pages.FirstPageInRange, pages.LastPageInRange = MinMax(pagesInRange)
	pages.PagesInRange = pagesInRange

	return &pages, nil
}

func (s *Paginator) getCurrentPageNumber() (int32, error) {
	return s.normalizePageNumber(s.CurrentPageNumber)
}

func (s *Paginator) getPagesInRange(lowerBound int32, upperBound int32) ([]int32, error) {
	var err error
	lowerBound, err = s.normalizePageNumber(lowerBound)

	if err != nil {
		return nil, err
	}

	upperBound, err = s.normalizePageNumber(upperBound)

	if err != nil {
		return nil, err
	}

	pages := make([]int32, upperBound-lowerBound+1)

	for pageNumber := lowerBound; pageNumber <= upperBound; pageNumber++ {
		pages[pageNumber-lowerBound] = pageNumber
	}

	return pages, nil
}

func (s *Paginator) normalizePageNumber(pageNumber int32) (int32, error) {
	if pageNumber < 1 {
		pageNumber = 1
	}

	pageCount, err := s.Count()
	if err != nil {
		return 0, err
	}

	if pageCount > 0 && pageNumber > pageCount {
		pageNumber = pageCount
	}

	return pageNumber, nil
}

func (s *Paginator) GetItemsByPage(pageNumber int32) (*goqu.SelectDataset, error) {
	var err error
	pageNumber, err = s.normalizePageNumber(pageNumber)

	if err != nil {
		return nil, err
	}

	offset := (pageNumber - 1) * s.getItemCountPerPage()
	ds := *s.SQLSelect

	return ds.Offset(uint(offset)).Limit(uint(s.getItemCountPerPage())), nil
}

func (s *Paginator) GetCurrentItems() (*goqu.SelectDataset, error) {
	pageNumber, err := s.getCurrentPageNumber()
	if err != nil {
		return nil, err
	}

	return s.GetItemsByPage(pageNumber)
}

func (s *Paginator) GetTotalItemCount() (int32, error) {
	var err error
	if !s.itemCountCalculated {
		s.itemCount, err = s.calculateCount()
		if err != nil {
			return 0, err
		}

		s.itemCountCalculated = true
	}

	return s.itemCount, nil
}
