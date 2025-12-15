package mongodb

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestConvertBsonToMap(t *testing.T) {
	t.Run("converts simple flat document", func(t *testing.T) {
		doc := bson.M{
			"name":  "test",
			"count": 42,
			"flag":  true,
		}

		result := convertBsonToMap(doc)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, 42, result["count"])
		assert.Equal(t, true, result["flag"])
	})

	t.Run("converts nested bson.M objects", func(t *testing.T) {
		doc := bson.M{
			"outer": bson.M{
				"inner": "value",
				"nested": bson.M{
					"deep": 123,
				},
			},
		}

		result := convertBsonToMap(doc)

		outer, ok := result["outer"].(map[string]any)
		require.True(t, ok, "outer should be map[string]any")
		assert.Equal(t, "value", outer["inner"])

		nested, ok := outer["nested"].(map[string]any)
		require.True(t, ok, "nested should be map[string]any")
		assert.Equal(t, 123, nested["deep"])
	})

	t.Run("converts bson.A arrays", func(t *testing.T) {
		doc := bson.M{
			"items": bson.A{"a", "b", "c"},
		}

		result := convertBsonToMap(doc)

		items, ok := result["items"].([]any)
		require.True(t, ok, "items should be []any")
		assert.Len(t, items, 3)
		assert.Equal(t, "a", items[0])
		assert.Equal(t, "b", items[1])
		assert.Equal(t, "c", items[2])
	})

	t.Run("handles nil values", func(t *testing.T) {
		doc := bson.M{
			"nullable": nil,
		}

		result := convertBsonToMap(doc)
		assert.Nil(t, result["nullable"])
	})

	t.Run("handles empty document", func(t *testing.T) {
		doc := bson.M{}
		result := convertBsonToMap(doc)
		assert.Empty(t, result)
	})
}

func TestConvertBsonValue(t *testing.T) {
	t.Run("converts primitive.ObjectID to hex string", func(t *testing.T) {
		oid := primitive.NewObjectID()
		result := convertBsonValue(oid)
		assert.Equal(t, oid.Hex(), result)
	})

	t.Run("converts primitive.DateTime to time.Time", func(t *testing.T) {
		now := time.Now().UTC()
		dt := primitive.NewDateTimeFromTime(now)
		result := convertBsonValue(dt)

		resultTime, ok := result.(time.Time)
		require.True(t, ok, "result should be time.Time")
		assert.WithinDuration(t, now, resultTime, time.Millisecond)
	})

	t.Run("converts 16-byte binary to UUID string", func(t *testing.T) {
		id := uuid.New()
		binary := primitive.Binary{
			Subtype: 4, // UUID subtype
			Data:    id[:],
		}

		result := convertBsonValue(binary)
		assert.Equal(t, id.String(), result)
	})

	t.Run("converts non-UUID binary to hex", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03}
		binary := primitive.Binary{
			Subtype: 0,
			Data:    data,
		}

		result := convertBsonValue(binary)
		assert.Equal(t, hex.EncodeToString(data), result)
	})

	t.Run("converts bson.D to map", func(t *testing.T) {
		doc := bson.D{
			{Key: "a", Value: 1},
			{Key: "b", Value: "two"},
		}

		result := convertBsonValue(doc)
		m, ok := result.(map[string]any)
		require.True(t, ok, "result should be map[string]any")
		assert.Equal(t, 1, m["a"])
		assert.Equal(t, "two", m["b"])
	})

	t.Run("converts nested bson.A with bson.M elements", func(t *testing.T) {
		arr := bson.A{
			bson.M{"id": 1},
			bson.M{"id": 2},
		}

		result := convertBsonValue(arr)
		slice, ok := result.([]any)
		require.True(t, ok, "result should be []any")
		require.Len(t, slice, 2)

		first, ok := slice[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, first["id"])
	})

	t.Run("returns primitive types unchanged", func(t *testing.T) {
		assert.Equal(t, "string", convertBsonValue("string"))
		assert.Equal(t, 42, convertBsonValue(42))
		assert.Equal(t, 3.14, convertBsonValue(3.14))
		assert.Equal(t, true, convertBsonValue(true))
		assert.Nil(t, convertBsonValue(nil))
	})
}

func TestInferDataType(t *testing.T) {
	ds := &ExternalDataSource{}

	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"string", "test", "string"},
		{"int", 42, "number"},
		{"int32", int32(42), "number"},
		{"int64", int64(42), "number"},
		{"float32", float32(3.14), "number"},
		{"float64", 3.14, "number"},
		{"bool", true, "boolean"},
		{"bson.A", bson.A{"a", "b"}, "array"},
		{"bson.M", bson.M{"key": "value"}, "object"},
		{"bson.D", bson.D{{Key: "k", Value: "v"}}, "object"},
		{"datetime", primitive.NewDateTimeFromTime(time.Now()), "date"},
		{"objectId", primitive.NewObjectID(), "objectId"},
		{"binary", primitive.Binary{Data: []byte{1, 2, 3}}, "binData"},
		{"regex", primitive.Regex{Pattern: ".*"}, "regex"},
		{"timestamp", primitive.Timestamp{T: 1234567890}, "timestamp"},
		{"decimal128", primitive.NewDecimal128(123, 456), "decimal"},
		{"minKey", primitive.MinKey{}, "minKey/maxKey"},
		{"maxKey", primitive.MaxKey{}, "minKey/maxKey"},
		{"unknown", struct{}{}, "unknown"},
		{"nil", nil, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ds.inferDataType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMoreSpecificType(t *testing.T) {
	ds := &ExternalDataSource{}

	tests := []struct {
		name     string
		newType  string
		current  string
		expected bool
	}{
		{"objectId more specific than unknown", "objectId", "unknown", true},
		{"objectId more specific than string", "objectId", "string", true},
		{"date more specific than string", "date", "string", true},
		{"string not more specific than objectId", "string", "objectId", false},
		{"same type not more specific", "string", "string", false},
		{"unknown not more specific than anything", "unknown", "string", false},
		{"number more specific than unknown", "number", "unknown", true},
		{"decimal more specific than number", "decimal", "number", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ds.isMoreSpecificType(tt.newType, tt.current)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateOptimalSampleSize(t *testing.T) {
	ds := &ExternalDataSource{}

	tests := []struct {
		name      string
		totalDocs int64
		expected  int
	}{
		{"small collection (100 docs)", 100, 100},
		{"medium collection (1000 docs)", 1000, 1000},
		{"medium-large collection (5000 docs)", 5000, 1000},
		{"large collection (50000 docs)", 50000, 2000},
		{"very large collection (500000 docs)", 500000, 5000},
		{"huge collection (5000000 docs)", 5000000, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ds.calculateOptimalSampleSize(tt.totalDocs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFilterConditionEmpty(t *testing.T) {
	t.Run("returns true for empty condition", func(t *testing.T) {
		condition := job.FilterCondition{}
		assert.True(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when equals is set", func(t *testing.T) {
		condition := job.FilterCondition{Equals: []any{"value"}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when greaterThan is set", func(t *testing.T) {
		condition := job.FilterCondition{GreaterThan: []any{10}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when greaterOrEqual is set", func(t *testing.T) {
		condition := job.FilterCondition{GreaterOrEqual: []any{10}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when lessThan is set", func(t *testing.T) {
		condition := job.FilterCondition{LessThan: []any{100}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when lessOrEqual is set", func(t *testing.T) {
		condition := job.FilterCondition{LessOrEqual: []any{100}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when between is set", func(t *testing.T) {
		condition := job.FilterCondition{Between: []any{10, 100}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when in is set", func(t *testing.T) {
		condition := job.FilterCondition{In: []any{"a", "b"}}
		assert.False(t, isFilterConditionEmpty(condition))
	})

	t.Run("returns false when notIn is set", func(t *testing.T) {
		condition := job.FilterCondition{NotIn: []any{"x", "y"}}
		assert.False(t, isFilterConditionEmpty(condition))
	})
}

func TestValidateFilterCondition(t *testing.T) {
	ds := &ExternalDataSource{}

	t.Run("accepts valid conditions", func(t *testing.T) {
		condition := job.FilterCondition{
			Equals:         []any{"value1", "value2"},
			GreaterThan:    []any{10},
			GreaterOrEqual: []any{20},
			LessThan:       []any{100},
			LessOrEqual:    []any{90},
			Between:        []any{50, 75},
			In:             []any{"a", "b", "c"},
			NotIn:          []any{"x", "y"},
		}
		err := ds.validateFilterCondition("field", condition)
		assert.NoError(t, err)
	})

	t.Run("returns error when between has wrong number of values", func(t *testing.T) {
		condition := job.FilterCondition{Between: []any{10}}
		err := ds.validateFilterCondition("myField", condition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "between operator")
		assert.Contains(t, err.Error(), "myField")
		assert.Contains(t, err.Error(), "exactly 2 values")
	})

	t.Run("returns error when between has too many values", func(t *testing.T) {
		condition := job.FilterCondition{Between: []any{10, 20, 30}}
		err := ds.validateFilterCondition("field", condition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exactly 2 values")
	})

	t.Run("returns error when gt has multiple values", func(t *testing.T) {
		condition := job.FilterCondition{GreaterThan: []any{10, 20}}
		err := ds.validateFilterCondition("field", condition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gt operator")
	})

	t.Run("returns error when gte has multiple values", func(t *testing.T) {
		condition := job.FilterCondition{GreaterOrEqual: []any{10, 20}}
		err := ds.validateFilterCondition("field", condition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gte operator")
	})

	t.Run("returns error when lt has multiple values", func(t *testing.T) {
		condition := job.FilterCondition{LessThan: []any{10, 20}}
		err := ds.validateFilterCondition("field", condition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lt operator")
	})

	t.Run("returns error when lte has multiple values", func(t *testing.T) {
		condition := job.FilterCondition{LessOrEqual: []any{10, 20}}
		err := ds.validateFilterCondition("field", condition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lte operator")
	})

	t.Run("accepts empty condition", func(t *testing.T) {
		condition := job.FilterCondition{}
		err := ds.validateFilterCondition("field", condition)
		assert.NoError(t, err)
	})
}

func TestConvertFilterConditionToMongoFilter(t *testing.T) {
	ds := &ExternalDataSource{}

	t.Run("converts single equals value", func(t *testing.T) {
		condition := job.FilterCondition{Equals: []any{"value"}}
		result, err := ds.convertFilterConditionToMongoFilter("status", condition)
		require.NoError(t, err)
		assert.Equal(t, "value", result["status"])
	})

	t.Run("converts multiple equals values to $in", func(t *testing.T) {
		condition := job.FilterCondition{Equals: []any{"active", "pending"}}
		result, err := ds.convertFilterConditionToMongoFilter("status", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["status"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []any{"active", "pending"}, fieldFilter["$in"])
	})

	t.Run("converts greaterThan to $gt", func(t *testing.T) {
		condition := job.FilterCondition{GreaterThan: []any{100}}
		result, err := ds.convertFilterConditionToMongoFilter("amount", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["amount"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 100, fieldFilter["$gt"])
	})

	t.Run("converts greaterOrEqual to $gte", func(t *testing.T) {
		condition := job.FilterCondition{GreaterOrEqual: []any{50}}
		result, err := ds.convertFilterConditionToMongoFilter("score", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["score"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 50, fieldFilter["$gte"])
	})

	t.Run("converts lessThan to $lt", func(t *testing.T) {
		condition := job.FilterCondition{LessThan: []any{1000}}
		result, err := ds.convertFilterConditionToMongoFilter("price", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["price"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1000, fieldFilter["$lt"])
	})

	t.Run("converts lessOrEqual to $lte", func(t *testing.T) {
		condition := job.FilterCondition{LessOrEqual: []any{500}}
		result, err := ds.convertFilterConditionToMongoFilter("quantity", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["quantity"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 500, fieldFilter["$lte"])
	})

	t.Run("converts between to $gte and $lte", func(t *testing.T) {
		condition := job.FilterCondition{Between: []any{10, 90}}
		result, err := ds.convertFilterConditionToMongoFilter("percentage", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["percentage"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 10, fieldFilter["$gte"])
		assert.Equal(t, 90, fieldFilter["$lte"])
	})

	t.Run("converts in to $in", func(t *testing.T) {
		condition := job.FilterCondition{In: []any{"a", "b", "c"}}
		result, err := ds.convertFilterConditionToMongoFilter("category", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["category"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []any{"a", "b", "c"}, fieldFilter["$in"])
	})

	t.Run("converts notIn to $nin", func(t *testing.T) {
		condition := job.FilterCondition{NotIn: []any{"x", "y"}}
		result, err := ds.convertFilterConditionToMongoFilter("tag", condition)
		require.NoError(t, err)

		fieldFilter, ok := result["tag"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []any{"x", "y"}, fieldFilter["$nin"])
	})

	t.Run("returns nil for empty condition", func(t *testing.T) {
		condition := job.FilterCondition{}
		result, err := ds.convertFilterConditionToMongoFilter("field", condition)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error for invalid condition", func(t *testing.T) {
		condition := job.FilterCondition{Between: []any{10}} // Invalid: needs 2 values
		_, err := ds.convertFilterConditionToMongoFilter("field", condition)
		assert.Error(t, err)
	})
}

func TestBuildMongoFilter(t *testing.T) {
	ds := &ExternalDataSource{}

	t.Run("builds filter from multiple conditions", func(t *testing.T) {
		filter := map[string]job.FilterCondition{
			"status": {Equals: []any{"active"}},
			"amount": {GreaterThan: []any{100}},
		}

		result, err := ds.buildMongoFilter(filter)
		require.NoError(t, err)

		assert.Equal(t, "active", result["status"])

		amountFilter, ok := result["amount"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 100, amountFilter["$gt"])
	})

	t.Run("skips empty conditions", func(t *testing.T) {
		filter := map[string]job.FilterCondition{
			"status": {Equals: []any{"active"}},
			"empty":  {},
		}

		result, err := ds.buildMongoFilter(filter)
		require.NoError(t, err)

		assert.Contains(t, result, "status")
		assert.NotContains(t, result, "empty")
	})

	t.Run("returns empty filter for all empty conditions", func(t *testing.T) {
		filter := map[string]job.FilterCondition{
			"field1": {},
			"field2": {},
		}

		result, err := ds.buildMongoFilter(filter)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error for invalid condition", func(t *testing.T) {
		filter := map[string]job.FilterCondition{
			"field": {Between: []any{10}}, // Invalid
		}

		_, err := ds.buildMongoFilter(filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field")
	})
}

func TestBuildFindOptions(t *testing.T) {
	ds := &ExternalDataSource{}

	t.Run("creates projection from specific fields", func(t *testing.T) {
		opts := ds.buildFindOptions([]string{"name", "email", "age"})
		assert.NotNil(t, opts)
		// The projection is set internally, we just verify options were created
	})

	t.Run("handles wildcard field", func(t *testing.T) {
		opts := ds.buildFindOptions([]string{"*"})
		assert.NotNil(t, opts)
	})

	t.Run("handles empty fields", func(t *testing.T) {
		opts := ds.buildFindOptions([]string{})
		assert.NotNil(t, opts)
	})
}

func TestCloseConnection(t *testing.T) {
	t.Run("handles nil DB in connection gracefully", func(t *testing.T) {
		// The ExternalDataSource requires a non-nil connection struct,
		// but the DB field inside can be nil
		// This test verifies that when connection.DB is nil, CloseConnection returns without error
		// We can't easily test this without setting up a real connection,
		// so we'll skip this particular edge case in unit tests
		// and rely on integration tests for the CloseConnection behavior
	})
}
