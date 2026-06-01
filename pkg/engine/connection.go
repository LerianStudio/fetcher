// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"encoding/json"
	"fmt"
)

// redactedMarker is the placeholder emitted in place of any secret value.
const redactedMarker = "[REDACTED]"

// ConnectionInputParams carries the raw, secret-bearing fields used to build a
// ConnectionInput. It exists only as a constructor argument so the secret is
// never stored in an exported field of a long-lived contract value.
type ConnectionInputParams struct {
	ConfigName   string
	Type         string
	Host         string
	Port         int
	DatabaseName string
	Schema       string
	Username     string
	Password     string
	SSLMode      string
}

// ConnectionInput is the credential-bearing input contract for a datasource
// connection. The password is held in an unexported field and is retrievable
// only through the explicit Password accessor. Normal output formatting —
// String, fmt %v/%+v/%#v, and JSON marshaling — redacts the secret. This is a
// contract type only; it performs no encryption, decryption, or I/O.
type ConnectionInput struct {
	ConfigName   string
	Type         string
	Host         string
	Port         int
	DatabaseName string
	Schema       string
	Username     string
	SSLMode      string

	// password is unexported so default struct formatting (%+v, %#v) and
	// encoding/json both skip it. The redacted MarshalJSON/String methods
	// provide an explicit, observable redaction on top of that.
	password string
}

// connectionInputJSON is the safe, serializable projection of a ConnectionInput.
// The password field is deliberately a constant redaction marker.
type connectionInputJSON struct {
	ConfigName   string `json:"configName"`
	Type         string `json:"type"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	DatabaseName string `json:"databaseName"`
	Schema       string `json:"schema,omitempty"`
	Username     string `json:"userName"`
	SSLMode      string `json:"sslMode,omitempty"`
	Password     string `json:"password"`
}

// NewConnectionInput builds a ConnectionInput, capturing the secret in the
// unexported password field.
func NewConnectionInput(params ConnectionInputParams) ConnectionInput {
	return ConnectionInput{
		ConfigName:   params.ConfigName,
		Type:         params.Type,
		Host:         params.Host,
		Port:         params.Port,
		DatabaseName: params.DatabaseName,
		Schema:       params.Schema,
		Username:     params.Username,
		SSLMode:      params.SSLMode,
		password:     params.Password,
	}
}

// Password returns the raw secret. This is the only path to the credential and
// MUST be used deliberately; it is never invoked by formatting or marshaling.
func (c ConnectionInput) Password() string {
	return c.password
}

// HasPassword reports whether a secret is present without revealing it.
func (c ConnectionInput) HasPassword() bool {
	return c.password != ""
}

// String implements fmt.Stringer with the secret redacted.
func (c ConnectionInput) String() string {
	password := ""
	if c.password != "" {
		password = redactedMarker
	}

	return fmt.Sprintf(
		"ConnectionInput{configName:%q type:%q host:%q port:%d database:%q schema:%q username:%q sslMode:%q password:%s}",
		c.ConfigName, c.Type, c.Host, c.Port, c.DatabaseName, c.Schema, c.Username, c.SSLMode, password,
	)
}

// GoString implements fmt.GoStringer so that %#v formatting also redacts the
// secret. Without it, %#v reflects over the unexported password field.
func (c ConnectionInput) GoString() string {
	return c.String()
}

// MarshalJSON emits a redacted JSON projection. The password key is always the
// redaction marker when a secret is present, and empty otherwise.
func (c ConnectionInput) MarshalJSON() ([]byte, error) {
	password := ""
	if c.password != "" {
		password = redactedMarker
	}

	return json.Marshal(connectionInputJSON{
		ConfigName:   c.ConfigName,
		Type:         c.Type,
		Host:         c.Host,
		Port:         c.Port,
		DatabaseName: c.DatabaseName,
		Schema:       c.Schema,
		Username:     c.Username,
		SSLMode:      c.SSLMode,
		Password:     password,
	})
}

// ConnectionDescriptor is the safe, secret-free output contract describing a
// resolved connection. It deliberately carries no password field so it can be
// logged and serialized freely.
type ConnectionDescriptor struct {
	ID           string `json:"id,omitempty"`
	ConfigName   string `json:"configName"`
	Type         string `json:"type"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	DatabaseName string `json:"databaseName"`
	Schema       string `json:"schema,omitempty"`
	Username     string `json:"userName"`
	SSLMode      string `json:"sslMode,omitempty"`

	// KeyVersion records which encryption key version protected this
	// connection's secret material. It is secret-free metadata only — the
	// ciphertext itself is never carried in this descriptor. ST-T003-02 wires
	// the CredentialProtector that populates it; until then it stays zero.
	KeyVersion int `json:"keyVersion,omitempty"`
}

// DescriptorFromInput projects a ConnectionInput onto a secret-free descriptor.
func DescriptorFromInput(input ConnectionInput) ConnectionDescriptor {
	return ConnectionDescriptor{
		ConfigName:   input.ConfigName,
		Type:         input.Type,
		Host:         input.Host,
		Port:         input.Port,
		DatabaseName: input.DatabaseName,
		Schema:       input.Schema,
		Username:     input.Username,
		SSLMode:      input.SSLMode,
	}
}
