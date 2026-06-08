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

	// HostAttributes is an OPAQUE host payload. The Engine carries it from input
	// to descriptor to the ConnectionStore and back without reading, validating,
	// or scoping on a single key — exactly like ExtractionPlan.Metadata. It exists
	// so a host with a richer persisted record (e.g. the Manager's ProductName,
	// full SSL CA/Cert/Key, uuid identity, metadata, timestamps) can round-trip
	// that record through the Engine's (tenantID, configName)-scoped connection
	// ops WITHOUT those host fields becoming Engine scoping dimensions and WITHOUT
	// field loss. It is secret-free by host contract; the Engine never serializes
	// it outward (see ConnectionDescriptor.HostAttributes, json:"-").
	HostAttributes map[string]any
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

	// hostAttributes is the OPAQUE host payload (see ConnectionInputParams). It
	// is unexported so default formatting and JSON marshaling skip it; the Engine
	// only forwards it through DescriptorFromInput. It is never interpreted here.
	hostAttributes map[string]any

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
		ConfigName:     params.ConfigName,
		Type:           params.Type,
		Host:           params.Host,
		Port:           params.Port,
		DatabaseName:   params.DatabaseName,
		Schema:         params.Schema,
		Username:       params.Username,
		SSLMode:        params.SSLMode,
		hostAttributes: params.HostAttributes,
		password:       params.Password,
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

	// HostAttributes is the OPAQUE host payload the Engine carries through its
	// (tenantID, configName)-scoped connection ops without interpreting any key
	// (see ConnectionInputParams.HostAttributes). It is json:"-" because it is
	// host-internal: it travels input -> descriptor -> ConnectionStore and back,
	// but is never part of the descriptor's PUBLIC serialized output contract.
	// The Engine treats it as a black box — it neither scopes nor validates on
	// it, which is what keeps host concepts (ProductName, org) out of Engine
	// scope while still enabling a lossless rich-model round-trip.
	HostAttributes map[string]any `json:"-"`
}

// ConnectionListParams is the OPAQUE list-mechanics carrier the Engine forwards
// to ConnectionStore.ListPaged without interpreting a single field. It exists so
// a host with a richer read surface than the Engine's flat List — the Manager's
// paginated, filtered, resolver-merged connection list — can carry its native
// filter/pagination object THROUGH the Engine's tenant-scoped read authority
// without the Engine learning what a page, a limit, a product filter, or a sort
// key is. The Engine enforces ONLY tenant scope, then delegates; it never reads,
// validates, paginates, sorts, or filters on this value.
//
// Filter holds the host's native list parameters (e.g. the Manager's
// net/http.QueryHeader) as an opaque any. The adapter on the host side
// type-asserts it back to its concrete type; the Engine treats it as a black box,
// exactly as it treats ConnectionDescriptor.HostAttributes.
type ConnectionListParams struct {
	Filter any
}

// ConnectionPage is the OPAQUE result the Engine returns from ListConnectionsPaged.
// It carries the page of secret-free descriptors the store produced plus the
// host's total count, WITHOUT the Engine interpreting either: the Engine does not
// compute Total, does not slice Items, and does not re-order them. It returns
// verbatim whatever ConnectionStore.ListPaged produced, so the host's exact
// pagination behavior (page size, total math, ordering) remains the host's
// contract. Each descriptor's HostAttributes carries the host's rich record for a
// lossless round-trip, identical to the single-record paths.
type ConnectionPage struct {
	Items []ConnectionDescriptor
	Total int64
}

// DescriptorFromInput projects a ConnectionInput onto a secret-free descriptor.
func DescriptorFromInput(input ConnectionInput) ConnectionDescriptor {
	return ConnectionDescriptor{
		ConfigName:     input.ConfigName,
		Type:           input.Type,
		Host:           input.Host,
		Port:           input.Port,
		DatabaseName:   input.DatabaseName,
		Schema:         input.Schema,
		Username:       input.Username,
		SSLMode:        input.SSLMode,
		HostAttributes: input.hostAttributes,
	}
}
