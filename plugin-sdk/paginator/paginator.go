package paginator

// OffsetPaginator provides helper calculations and state tracking for limit/offset based APIs.
type OffsetPaginator struct {
	Limit      int
	Offset     int
	TotalCount int // -1 if unknown
}

// NewOffsetPaginator creates a new paginator with limit and initial offset.
func NewOffsetPaginator(limit, initialOffset int) *OffsetPaginator {
	if limit <= 0 {
		limit = 20
	}
	if initialOffset < 0 {
		initialOffset = 0
	}
	return &OffsetPaginator{
		Limit:      limit,
		Offset:     initialOffset,
		TotalCount: -1,
	}
}

// SetTotal sets the known total item count.
func (p *OffsetPaginator) SetTotal(total int) {
	p.TotalCount = total
}

// Page returns the 1-based page number corresponding to current Offset.
func (p *OffsetPaginator) Page() int {
	if p.Limit <= 0 {
		return 1
	}
	return (p.Offset / p.Limit) + 1
}

// TotalPages returns total page count when TotalCount is known, or -1 if unknown.
func (p *OffsetPaginator) TotalPages() int {
	if p.TotalCount < 0 || p.Limit <= 0 {
		return -1
	}
	if p.TotalCount == 0 {
		return 0
	}
	return (p.TotalCount + p.Limit - 1) / p.Limit
}

// HasNextPage determines whether subsequent pages exist given the count of items in the current page.
func (p *OffsetPaginator) HasNextPage(itemsInCurrentPage int) bool {
	if itemsInCurrentPage <= 0 {
		return false
	}
	if p.TotalCount >= 0 {
		return p.Offset+itemsInCurrentPage < p.TotalCount
	}
	return itemsInCurrentPage >= p.Limit
}

// NextOffset calculates the next offset value.
func (p *OffsetPaginator) NextOffset() int {
	return p.Offset + p.Limit
}

// Advance updates the current offset to the next page if items were returned.
// Returns true if there is a next page.
func (p *OffsetPaginator) Advance(itemsInCurrentPage int) bool {
	if !p.HasNextPage(itemsInCurrentPage) {
		return false
	}
	p.Offset = p.NextOffset()
	return true
}

// CursorPaginator tracks cursor-based pagination state.
type CursorPaginator struct {
	CurrentCursor string
	NextCursor    string
	HasMore       bool
}

// NewCursorPaginator creates a paginator starting with the specified initial cursor.
func NewCursorPaginator(initialCursor string) *CursorPaginator {
	return &CursorPaginator{
		CurrentCursor: initialCursor,
		HasMore:       true,
	}
}

// Update records the next cursor and availability indicator from an API response.
func (p *CursorPaginator) Update(nextCursor string, hasMore bool) {
	p.NextCursor = nextCursor
	p.HasMore = hasMore && nextCursor != ""
}

// Advance moves CurrentCursor to NextCursor. Returns true if pagination can continue.
func (p *CursorPaginator) Advance() bool {
	if !p.HasMore || p.NextCursor == "" {
		p.HasMore = false
		return false
	}
	p.CurrentCursor = p.NextCursor
	p.NextCursor = ""
	return true
}
