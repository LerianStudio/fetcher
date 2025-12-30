package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagination_SetItems(t *testing.T) {
	tests := []struct {
		name     string
		items    any
		expected any
	}{
		{
			name:     "set string slice",
			items:    []string{"item1", "item2", "item3"},
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "set integer slice",
			items:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "set struct slice",
			items:    []JobResponse{{Status: "pending"}, {Status: "completed"}},
			expected: []JobResponse{{Status: "pending"}, {Status: "completed"}},
		},
		{
			name:     "set nil items",
			items:    nil,
			expected: nil,
		},
		{
			name:     "set empty slice",
			items:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{}
			p.SetItems(tt.items)

			assert.Equal(t, tt.expected, p.Items)
		})
	}
}

func TestPagination_SetTotal(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		expected int
	}{
		{name: "set positive total", total: 100, expected: 100},
		{name: "set zero total", total: 0, expected: 0},
		{name: "set large total", total: 999999, expected: 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{}
			p.SetTotal(tt.total)

			assert.Equal(t, tt.expected, p.Total)
		})
	}
}

func TestPagination_Integration(t *testing.T) {
	t.Run("create pagination with all fields", func(t *testing.T) {
		p := &Pagination{
			Page:  1,
			Limit: 10,
		}

		items := []string{"item1", "item2", "item3"}
		p.SetItems(items)
		p.SetTotal(100)

		assert.Equal(t, items, p.Items)
		assert.Equal(t, 1, p.Page)
		assert.Equal(t, 10, p.Limit)
		assert.Equal(t, 100, p.Total)
	})

	t.Run("update existing pagination", func(t *testing.T) {
		p := &Pagination{
			Items: []string{"old1", "old2"},
			Page:  1,
			Limit: 5,
			Total: 50,
		}

		newItems := []string{"new1", "new2", "new3"}
		p.SetItems(newItems)
		p.SetTotal(75)

		assert.Equal(t, newItems, p.Items)
		assert.Equal(t, 1, p.Page)
		assert.Equal(t, 5, p.Limit)
		assert.Equal(t, 75, p.Total)
	})
}
