// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"sort"
)

// eagerCursor adapts a fully-materialized result map into a RowCursor. It is the
// bridge for connectors that fetch all rows up front (the historical Query
// shape) and have not yet been backed by a true DB-side cursor: they materialize
// the whole result, hand it to NewEagerCursor, and the engine drives it through
// the streaming contract identically to a real cursor.
//
// DETERMINISM IS LOAD-BEARING. The engine's streamed output and integrity digest
// are defined to be byte-identical across runs, so an eager cursor MUST iterate
// in a stable order regardless of Go's randomized map iteration: tables are
// visited in SORTED key order, and the rows within each table follow their slice
// order. The flattening is computed once at construction so the iteration order
// is frozen before the first Next.
type eagerCursor struct {
	// flat holds the rows pre-flattened into (table, row) pairs in deterministic
	// order. Holding the order here, not recomputed per Next, keeps iteration
	// stable and cheap.
	pairs []eagerRow
	// pos is the index of the CURRENT row + 1 (0 before the first Next). It is
	// valid (points at a real row via pos-1) only while exhausted is false.
	pos int
	// exhausted is set once Next has returned false, so Row reverts to the zero
	// value at end-of-stream rather than echoing the final row.
	exhausted bool
	// err holds a context cancellation observed in Next, surfaced via Err so a
	// caller can distinguish a cancelled stream from a clean end-of-stream. An
	// eager cursor has no streaming I/O, so this is the only error it can carry.
	err error
}

// eagerRow is one flattened (table, row) pair.
type eagerRow struct {
	table string
	row   map[string]any
}

// NewEagerCursor adapts a fully-materialized result map into a RowCursor.
// Used by adapters that fetch-all today; true DB-side streaming is a later
// optimization. Tables are iterated in SORTED key order and rows in slice order
// so the cursor's output is deterministic regardless of map-iteration order.
func NewEagerCursor(data map[string][]map[string]any) RowCursor {
	tables := make([]string, 0, len(data))
	for table := range data {
		tables = append(tables, table)
	}

	sort.Strings(tables)

	total := 0

	for _, rows := range data {
		total += len(rows)
	}

	pairs := make([]eagerRow, 0, total)

	for _, table := range tables {
		rows := data[table]
		for _, row := range rows {
			pairs = append(pairs, eagerRow{table: table, row: row})
		}
	}

	return &eagerCursor{pairs: pairs}
}

// Next advances to the next (table, row) pair. The only error it can report is
// context cancellation: an eager cursor has already materialized its rows, so
// there is no streaming I/O left to fail. It honors cancellation by stopping AND
// recording ctx.Err() so Err can surface it — a driving loop that cancels
// mid-iteration stops promptly and can tell the abort apart from a clean EOF.
func (c *eagerCursor) Next(ctx context.Context) bool {
	if c.err != nil {
		return false
	}

	if ctx.Err() != nil {
		c.err = ctx.Err()
		c.exhausted = true

		return false
	}

	if c.pos >= len(c.pairs) {
		c.exhausted = true
		return false
	}

	c.pos++

	return true
}

// Row returns the current (table, row) pair. It is valid only after Next
// returned true; calling it before the first Next or after Next returned false
// yields the zero row.
func (c *eagerCursor) Row() (string, map[string]any) {
	if c.exhausted || c.pos == 0 || c.pos > len(c.pairs) {
		return "", nil
	}

	pair := c.pairs[c.pos-1]

	return pair.table, pair.row
}

// Err returns the context cancellation observed by Next, if any. It is nil for a
// cursor that drained to a clean end-of-stream, so a caller can distinguish a
// completed extraction from one aborted mid-iteration.
func (c *eagerCursor) Err() error {
	return c.err
}

// Close releases the held rows. It is idempotent and double-close safe.
func (c *eagerCursor) Close(_ context.Context) error {
	c.pairs = nil
	c.pos = 0
	c.exhausted = true
	c.err = nil

	return nil
}
