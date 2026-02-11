package mongodb

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

// IsIndexConflictError checks if the error is a MongoDB index conflict error.
// IndexOptionsConflict is code 85, IndexKeySpecsConflict is code 86.
func IsIndexConflictError(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 85 || cmdErr.Code == 86
	}

	return false
}
