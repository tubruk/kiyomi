package paginator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOffsetPaginator(t *testing.T) {
	p := NewOffsetPaginator(10, 0)
	assert.Equal(t, 10, p.Limit)
	assert.Equal(t, 0, p.Offset)
	assert.Equal(t, 1, p.Page())
	assert.Equal(t, -1, p.TotalPages())

	// When limit is 10 and 10 items returned (total unknown)
	assert.True(t, p.HasNextPage(10))
	assert.True(t, p.Advance(10))
	assert.Equal(t, 10, p.Offset)
	assert.Equal(t, 2, p.Page())

	// Last page returns 5 items (< limit)
	assert.False(t, p.HasNextPage(5))
	assert.False(t, p.Advance(5))
	assert.Equal(t, 10, p.Offset) // stays at current

	// Total count known
	p2 := NewOffsetPaginator(10, 0)
	p2.SetTotal(25)
	assert.Equal(t, 3, p2.TotalPages())
	assert.True(t, p2.HasNextPage(10))
	p2.Advance(10) // offset = 10, page = 2
	assert.Equal(t, 2, p2.Page())
	assert.True(t, p2.HasNextPage(10))
	p2.Advance(10) // offset = 20, page = 3
	assert.Equal(t, 3, p2.Page())
	assert.False(t, p2.HasNextPage(5)) // 20 + 5 == 25 (reached total)

	// Zero total
	p3 := NewOffsetPaginator(10, 0)
	p3.SetTotal(0)
	assert.Equal(t, 0, p3.TotalPages())
	assert.False(t, p3.HasNextPage(0))
}

func TestCursorPaginator(t *testing.T) {
	p := NewCursorPaginator("initial_token")
	assert.Equal(t, "initial_token", p.CurrentCursor)
	assert.True(t, p.HasMore)

	p.Update("next_token_1", true)
	assert.True(t, p.Advance())
	assert.Equal(t, "next_token_1", p.CurrentCursor)

	p.Update("", false)
	assert.False(t, p.Advance())
	assert.False(t, p.HasMore)
}
