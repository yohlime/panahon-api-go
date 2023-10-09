package util

type PaginatedList[T any] struct {
	Page       int32 `json:"page"`
	PerPage    int32 `json:"per_page"`
	TotalPages int32 `json:"total_pages"`
	NextPage   int32 `json:"next_page"`
	PrevPage   int32 `json:"prev_page"`
	Count      int32 `json:"count"`
	Items      []T   `json:"items"`
}

func NewPaginatedList[T any](page, limit, count int32, items []T) PaginatedList[T] {
	p := PaginatedList[T]{
		Page:    page,
		PerPage: limit,
		Count:   count,
		Items:   items,
	}

	p.TotalPages = p.calculateTotalPages()
	p.setNextPage()
	p.setPrevPage()

	return p
}

func (p PaginatedList[T]) calculateTotalPages() int32 {
	if p.PerPage <= 0 {
		return 0
	}
	return (p.Count + p.PerPage - 1) / p.PerPage
}

func (p *PaginatedList[T]) setNextPage() {
	nextPage := p.Page + 1
	if nextPage <= p.TotalPages {
		p.NextPage = nextPage
	}
}

func (p *PaginatedList[T]) setPrevPage() {
	prevPage := p.Page - 1
	if prevPage >= 1 {
		p.PrevPage = prevPage
	}
}
