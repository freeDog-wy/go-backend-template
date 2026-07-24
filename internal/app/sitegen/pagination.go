package sitegen

import (
	"context"
	"fmt"
)

// loadAllPages reads a stable complete collection from a paginated public API.
// The consistency checks are a static-build concern, not part of cmsclient.
func loadAllPages[T any](ctx context.Context, getPage func(context.Context, int) ([]T, *pageMeta, error)) ([]T, error) {
	first, meta, err := getPage(ctx, 1)
	if err != nil {
		return nil, err
	}
	if err := validatePageMeta(meta, 1); err != nil {
		return nil, err
	}
	all := append([]T(nil), first...)
	for page := 2; page <= meta.TotalPages; page++ {
		items, current, err := getPage(ctx, page)
		if err != nil {
			return nil, err
		}
		if err := validatePageMeta(current, page); err != nil {
			return nil, err
		}
		if current.TotalPages != meta.TotalPages || current.Total != meta.Total {
			return nil, fmt.Errorf("pagination metadata changed while reading page %d", page)
		}
		all = append(all, items...)
	}
	if int64(len(all)) != meta.Total {
		return nil, fmt.Errorf("pagination returned %d items, expected %d", len(all), meta.Total)
	}
	return all, nil
}

func validatePageMeta(meta *pageMeta, requestedPage int) error {
	if meta == nil {
		return fmt.Errorf("pagination metadata is missing")
	}
	if meta.Page != requestedPage || meta.PerPage < 1 || meta.Total < 0 || meta.TotalPages < 0 {
		return fmt.Errorf("invalid pagination metadata for page %d", requestedPage)
	}
	expectedPages := 0
	if meta.Total > 0 {
		expectedPages = int((meta.Total + int64(meta.PerPage) - 1) / int64(meta.PerPage))
	}
	if meta.TotalPages != expectedPages {
		return fmt.Errorf("invalid pagination total_pages for page %d", requestedPage)
	}
	return nil
}
