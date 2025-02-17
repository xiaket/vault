// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package event

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"

	"github.com/hashicorp/eventlogger"

	"github.com/stretchr/testify/require"
)

// fakeJSONxAuditEvent will return a new fake event containing audit data based
// on the specified auditSubtype and logical.LogInput.
func fakeJSONxAuditEvent(t *testing.T, subtype auditSubtype, input *logical.LogInput) *eventlogger.Event {
	t.Helper()

	date := time.Date(2023, time.July, 11, 15, 49, 10, 0, time.Local)

	auditEvent, err := newAudit(
		WithID("123"),
		WithSubtype(string(subtype)),
		WithFormat(string(AuditFormatJSONx)),
		WithNow(date),
	)
	require.NoError(t, err)
	require.NotNil(t, auditEvent)
	require.Equal(t, "123", auditEvent.ID)
	require.Equal(t, "v0.1", auditEvent.Version)
	require.Equal(t, AuditFormatJSONx, auditEvent.RequiredFormat)
	require.Equal(t, subtype, auditEvent.Subtype)
	require.Equal(t, date, auditEvent.Timestamp)

	auditEvent.Data = input

	e := &eventlogger.Event{
		Type:      eventlogger.EventType(AuditType),
		CreatedAt: auditEvent.Timestamp,
		Formatted: make(map[string][]byte),
		Payload:   auditEvent,
	}

	return e
}

// TestNewAuditFormatterJSONx ensures we can create new AuditFormatterJSONx structs.
func TestNewAuditFormatterJSONx(t *testing.T) {
	f := NewAuditFormatterJSONx()
	require.NotNil(t, f)
}

// TestAuditFormatterJSONx_Reopen ensures that we do no get an error when calling Reopen.
func TestAuditFormatterJSONx_Reopen(t *testing.T) {
	require.NoError(t, NewAuditFormatterJSONx().Reopen())
}

// TestAuditFormatterJSONx_Type ensures that the node is a 'formatter' type.
func TestAuditFormatterJSONx_Type(t *testing.T) {
	require.Equal(t, eventlogger.NodeTypeFormatter, NewAuditFormatterJSONx().Type())
}

// TestAuditFormatterJSONx_Process attempts to run the Process method to convert
// pre-formatted JSON to XML (JSONx).
func TestAuditFormatterJSONx_Process(t *testing.T) {
	tests := map[string]struct {
		IsErrorExpected      bool
		ExpectedErrorMessage string
		Subtype              auditSubtype
		Data                 *logical.LogInput
	}{
		"request-no-formatted-json": {
			IsErrorExpected:      true,
			ExpectedErrorMessage: "event.(AuditFormatterJSONx).Process: pre-formatted JSON required but not found: invalid parameter",
			Subtype:              AuditRequest,
			Data:                 nil,
		},
		"response-no-formatted-json": {
			IsErrorExpected:      true,
			ExpectedErrorMessage: "event.(AuditFormatterJSONx).Process: pre-formatted JSON required but not found: invalid parameter",
			Subtype:              AuditResponse,
			Data:                 nil,
		},
		"request-basic-json": {
			IsErrorExpected: false,
			Subtype:         AuditRequest,
			Data:            &logical.LogInput{Type: "magic"},
		},
		"response-basic-json": {
			IsErrorExpected: false,
			Subtype:         AuditResponse,
			Data:            &logical.LogInput{Type: "magic"},
		},
	}

	for name, tc := range tests {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := fakeJSONxAuditEvent(t, tc.Subtype, tc.Data)
			require.NotNil(t, e)

			// If we have data specified, then encode it and store as a format.
			// This is faking the behavior of the JSON formatter node which is a
			// pre-req for JSONx formatter node.
			if tc.Data != nil {
				jsonBytes, err := jsonutil.EncodeJSON(tc.Data)
				require.NoError(t, err)
				require.NotNil(t, jsonBytes)
				e.FormattedAs(string(AuditFormatJSON), jsonBytes)
			}

			processed, err := NewAuditFormatterJSONx().Process(context.Background(), e)
			b, found := e.Format(string(AuditFormatJSONx))

			switch {
			case tc.IsErrorExpected:
				require.Error(t, err)
				require.EqualError(t, err, tc.ExpectedErrorMessage)
				require.Nil(t, processed)
				require.False(t, found)
				require.Nil(t, b)
			default:
				require.NoError(t, err)
				require.NotNil(t, processed)
				require.True(t, found)
				require.NotNil(t, b)
			}
		})
	}
}
