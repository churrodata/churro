// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package handlers

// HandlerWrapper ...
type HandlerWrapper struct {
	StatusText   string
	UserEmail    string
	ErrorText    string
	DatabaseType string
}

// Copy ....
func (u HandlerWrapper) Copy(ErrorText string) (a HandlerWrapper) {
	a.StatusText = u.StatusText
	a.UserEmail = u.UserEmail
	a.ErrorText = ErrorText
	a.DatabaseType = u.DatabaseType
	return a
}
