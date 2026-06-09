package fixtures

import (
	"context"
	"embed"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

//go:embed *.sql
var SQLFixtures embed.FS

// TestAccountIDs are consistent across all databases.
var TestAccountIDs = []string{
	"11111111-1111-1111-1111-111111111111",
	"22222222-2222-2222-2222-222222222222",
	"33333333-3333-3333-3333-333333333333",
}

// TestOrganizationID is used for all test operations.
const TestOrganizationID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

// Categories used in test data.
var Categories = []string{"salary", "groceries", "utilities", "entertainment", "travel"}

// GetPostgresInitSQL returns the PostgreSQL init script.
func GetPostgresInitSQL() (string, error) {
	data, err := SQLFixtures.ReadFile("postgres_init.sql")
	if err != nil {
		return "", fmt.Errorf("failed to read postgres_init.sql: %w", err)
	}

	return string(data), nil
}

// GetMySQLInitSQL returns the MySQL init script.
func GetMySQLInitSQL() (string, error) {
	data, err := SQLFixtures.ReadFile("mysql_init.sql")
	if err != nil {
		return "", fmt.Errorf("failed to read mysql_init.sql: %w", err)
	}

	return string(data), nil
}

// GetSQLServerInitSQL returns the SQL Server init script.
func GetSQLServerInitSQL() (string, error) {
	data, err := SQLFixtures.ReadFile("sqlserver_init.sql")
	if err != nil {
		return "", fmt.Errorf("failed to read sqlserver_init.sql: %w", err)
	}

	return string(data), nil
}

// GetOracleInitSQL returns the Oracle init script.
func GetOracleInitSQL() (string, error) {
	data, err := SQLFixtures.ReadFile("oracle_init.sql")
	if err != nil {
		return "", fmt.Errorf("failed to read oracle_init.sql: %w", err)
	}

	return string(data), nil
}

// MongoDBTransaction represents a transaction document.
type MongoDBTransaction struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	AccountID   string        `bson:"account_id"`
	Amount      float64       `bson:"amount"`
	Currency    string        `bson:"currency"`
	Type        string        `bson:"type"`
	Description string        `bson:"description"`
	Category    string        `bson:"category"`
	Status      string        `bson:"status"`
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}

// InitMongoDBExternal initializes MongoDB with Q4 2024 test data.
func InitMongoDBExternal(ctx context.Context, connectionString, database string) error {
	// Configure client with shorter timeouts for faster retry cycles
	clientOpts := options.Client().
		ApplyURI(connectionString).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(5 * time.Second)

	// Retry connection with backoff
	var (
		client *mongo.Client
		err    error
	)

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		client, err = mongo.Connect(clientOpts)
		if err != nil {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}

		// Verify connection with ping
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = client.Ping(pingCtx, nil)

		cancel()

		if err == nil {
			break
		}

		_ = client.Disconnect(ctx)

		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB after %d retries: %w", maxRetries, err)
	}

	defer func() { _ = client.Disconnect(ctx) }()

	db := client.Database(database)
	coll := db.Collection("transactions")

	// Drop existing data
	_ = coll.Drop(ctx)

	// Create indexes
	_, err = coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "account_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: 1}}},
		{Keys: bson.D{{Key: "category", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Q4 2024 test data (October-December)
	transactions := []any{
		// October 2024
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 1500.00, Currency: "USD", Type: "credit", Description: "Salary October", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 10, 5, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 190.00, Currency: "USD", Type: "debit", Description: "Grocery Store", Category: "groceries", Status: "completed", CreatedAt: time.Date(2024, 10, 12, 14, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[1], Amount: 2300.00, Currency: "USD", Type: "credit", Description: "Freelance Work", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 10, 8, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[1], Amount: 75.00, Currency: "USD", Type: "debit", Description: "Movie Theater", Category: "entertainment", Status: "completed", CreatedAt: time.Date(2024, 10, 20, 19, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 3500.00, Currency: "USD", Type: "credit", Description: "Monthly Salary", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 10, 1, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 500.00, Currency: "EUR", Type: "debit", Description: "Flight Tickets", Category: "travel", Status: "completed", CreatedAt: time.Date(2024, 10, 25, 16, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},

		// November 2024
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 1500.00, Currency: "USD", Type: "credit", Description: "Salary November", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 11, 5, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 110.00, Currency: "USD", Type: "debit", Description: "Gas Bill", Category: "utilities", Status: "completed", CreatedAt: time.Date(2024, 11, 15, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[1], Amount: 1800.00, Currency: "USD", Type: "credit", Description: "Project Payment", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 11, 12, 14, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[1], Amount: 200.00, Currency: "GBP", Type: "debit", Description: "Hotel Stay", Category: "travel", Status: "completed", CreatedAt: time.Date(2024, 11, 22, 12, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 3500.00, Currency: "USD", Type: "credit", Description: "Monthly Salary", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 11, 1, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 65.00, Currency: "USD", Type: "debit", Description: "Streaming Services", Category: "entertainment", Status: "completed", CreatedAt: time.Date(2024, 11, 18, 8, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},

		// December 2024
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 1500.00, Currency: "USD", Type: "credit", Description: "Salary December", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 12, 5, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 350.00, Currency: "USD", Type: "debit", Description: "Holiday Shopping", Category: "groceries", Status: "completed", CreatedAt: time.Date(2024, 12, 20, 15, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[1], Amount: 3000.00, Currency: "USD", Type: "credit", Description: "Year-End Bonus", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 12, 28, 16, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[1], Amount: 180.00, Currency: "USD", Type: "debit", Description: "Restaurant", Category: "entertainment", Status: "completed", CreatedAt: time.Date(2024, 12, 24, 20, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 3500.00, Currency: "USD", Type: "credit", Description: "Monthly Salary", Category: "salary", Status: "completed", CreatedAt: time.Date(2024, 12, 1, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 95.00, Currency: "USD", Type: "debit", Description: "Electric Bill", Category: "utilities", Status: "completed", CreatedAt: time.Date(2024, 12, 10, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},

		// Pending transactions
		MongoDBTransaction{AccountID: TestAccountIDs[0], Amount: 80.00, Currency: "USD", Type: "debit", Description: "Pending Order", Category: "groceries", Status: "pending", CreatedAt: time.Date(2024, 12, 30, 14, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
		MongoDBTransaction{AccountID: TestAccountIDs[2], Amount: 120.00, Currency: "USD", Type: "debit", Description: "Pending Subscription", Category: "entertainment", Status: "pending", CreatedAt: time.Date(2024, 12, 31, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Now()},
	}

	_, err = coll.InsertMany(ctx, transactions)
	if err != nil {
		return fmt.Errorf("failed to insert transactions: %w", err)
	}

	return nil
}

// ExpectedPostgresRecordCount returns expected record count for PostgreSQL.
func ExpectedPostgresRecordCount() int {
	return 26 // 24 completed + 2 pending (Q1 2024)
}

// ExpectedMySQLRecordCount returns expected record count for MySQL.
func ExpectedMySQLRecordCount() int {
	return 20 // 18 completed + 2 pending (Q2 2024)
}

// ExpectedSQLServerRecordCount returns expected record count for SQL Server.
func ExpectedSQLServerRecordCount() int {
	return 26 // 24 completed + 2 pending (Q3 2024)
}

// ExpectedOracleRecordCount returns expected record count for Oracle.
func ExpectedOracleRecordCount() int {
	return 26 // 24 completed + 2 pending (Q4 2023 - Q1 2024)
}

// ExpectedMongoDBRecordCount returns expected record count for MongoDB External.
func ExpectedMongoDBRecordCount() int {
	return 20 // 18 completed + 2 pending (Q4 2024)
}
