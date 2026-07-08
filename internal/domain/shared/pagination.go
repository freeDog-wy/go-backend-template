package shared

type PageQuery struct {
	Page    int
	PerPage int
}

type PageResult struct {
	Page    int
	PerPage int
	Total   int64
}

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

func NewPageQuery(page, perPage int) PageQuery {
	if page <= 0 {
		page = DefaultPage
	}
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return PageQuery{
		Page:    page,
		PerPage: perPage,
	}
}

func (q PageQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

func (r PageResult) TotalPages() int {
	if r.PerPage <= 0 || r.Total <= 0 {
		return 0
	}
	totalPages := int(r.Total) / r.PerPage
	if int(r.Total)%r.PerPage != 0 {
		totalPages++
	}
	return totalPages
}
