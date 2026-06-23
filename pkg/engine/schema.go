// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"sort"
	"time"
)

// FieldSnapshot describes a single field/column in a table snapshot.
type FieldSnapshot struct {
	Name string `json:"name"`
}

// TableSnapshot describes one table or collection and its known fields.
type TableSnapshot struct {
	// Name is the qualified table/collection name (e.g. "schema.table" or
	// "database.collection").
	Name string `json:"name"`
	// Fields enumerates the known field names of the table.
	Fields []string `json:"fields"`
}

// SchemaSnapshot is the read-only schema contract describing the tables and
// fields available on a datasource at a point in time. It is a pure data
// contract: it performs no caching, querying, or tenant scoping. The zero
// value is safe and reports an empty schema.
type SchemaSnapshot struct {
	// ConfigName identifies the datasource the snapshot describes.
	ConfigName string `json:"configName"`
	// Tables enumerates the snapshot's tables.
	Tables []TableSnapshot `json:"tables"`
	// CapturedAt records when the snapshot was taken (optional).
	CapturedAt time.Time `json:"capturedAt,omitempty"`
}

// HasTable reports whether the snapshot contains the named table.
func (s SchemaSnapshot) HasTable(name string) bool {
	for _, table := range s.Tables {
		if table.Name == name {
			return true
		}
	}

	return false
}

// TableNames returns the snapshot's table names sorted for stable output.
func (s SchemaSnapshot) TableNames() []string {
	if len(s.Tables) == 0 {
		return nil
	}

	names := make([]string, 0, len(s.Tables))
	for _, table := range s.Tables {
		names = append(names, table.Name)
	}

	sort.Strings(names)

	return names
}
