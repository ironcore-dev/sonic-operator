// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	agent "github.com/ironcore-dev/switch-operator/agent/types"
)

const (
	CLIENT_ERROR = 1
	SERVER_ERROR = 2

	BAD_REQUEST = 101
	// General-purpose errors
	NOT_FOUND            = 201
	ALREADY_EXISTS       = 202
	REDIS_HSET_FAIL      = 203
	REDIS_HGET_FAIL      = 204
	REDIS_KEY_CHECK_FAIL = 205
)

func NewErrorStatus(code uint32, message string) *agent.Status {
	return &agent.Status{
		Code:    code,
		Message: message,
	}
}
