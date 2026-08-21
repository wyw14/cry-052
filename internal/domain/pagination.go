package domain

import (
	"fmt"
	"strings"
)

type PageRequest struct {
	Page    int
	Size    int
	Sort    string
	Filters map[string]string
}

type Page[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

func (p PageRequest) Normalize(allowedSort, allowedFilters map[string]struct{}) (PageRequest, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 {
		p.Size = 20
	}
	if p.Size > 100 {
		p.Size = 100
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if _, ok := allowedSort[p.Sort]; !ok {
		return PageRequest{}, NewValidationError("list query is invalid", FieldViolation{Field: "sort", Message: fmt.Sprintf("unsupported sort %q", p.Sort)})
	}
	for key := range p.Filters {
		if _, ok := allowedFilters[strings.ToLower(key)]; !ok {
			return PageRequest{}, NewValidationError("list query is invalid", FieldViolation{Field: "filter_" + key, Message: fmt.Sprintf("unsupported filter %q", key)})
		}
	}
	return p, nil
}

func Paginate[T any](items []T, req PageRequest) Page[T] {
	start := (req.Page - 1) * req.Size
	if start > len(items) {
		start = len(items)
	}
	end := start + req.Size
	if end > len(items) {
		end = len(items)
	}
	return Page[T]{Items: append([]T(nil), items[start:end]...), Page: req.Page, Size: req.Size, Total: len(items)}
}
