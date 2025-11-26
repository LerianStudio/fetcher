package connection

import (
	"strings"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"
)

type SSLInput struct {
	Mode string  `json:"mode"`
	CA   *string `json:"ca"`
	Cert *string `json:"cert"`
	Key  *string `json:"key"`
}

type ConnectionInput struct {
	ConfigName   string    `json:"configName"`
	Type         string    `json:"type"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	DatabaseName string    `json:"databaseName"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	SSL          *SSLInput `json:"ssl,omitempty"`
}

type ListConnectionsInput struct {
	Page         int
	Limit        int
	SortOrder    string
	Type         string
	ConfigName   string
	Host         string
	DatabaseName string
	CreatedAt    string
}

func SSLInputToModel(in *SSLInput) *domainConn.SSLConfig {
	if in == nil {
		return nil
	}

	ssl := &domainConn.SSLConfig{
		Mode: strings.TrimSpace(in.Mode),
	}
	if in.CA != nil {
		ssl.CA = *in.CA
	}
	if in.Cert != nil {
		ssl.Cert = *in.Cert
	}
	if in.Key != nil {
		ssl.Key = *in.Key
	}
	return ssl
}

func ValidationError(msg string) error {
	return pkg.ValidationError{
		EntityType: "connection",
		Title:      "Validation Error",
		Code:       constant.ErrBadRequest.Error(),
		Message:    msg,
	}
}

func NotFoundError() error {
	return pkg.EntityNotFoundError{
		EntityType: "connection",
		Code:       constant.ErrEntityNotFound.Error(),
		Title:      "Entity Not Found",
		Message:    "connection not found",
	}
}

func ConflictError(msg string) error {
	return pkg.EntityConflictError{
		EntityType: "connection",
		Code:       constant.ErrEntityConflict.Error(),
		Title:      "Conflict",
		Message:    msg,
	}
}
