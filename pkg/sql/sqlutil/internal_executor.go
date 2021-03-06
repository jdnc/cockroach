// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package sqlutil

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
)

// InternalExecutor is meant to be used by layers below SQL in the system that
// nevertheless want to execute SQL queries (presumably against system tables).
// It is extracted in this "sql/util" package to avoid circular references and
// is implemented by *sql.InternalExecutor.
type InternalExecutor interface {
	// ExecuteStatementInTransaction executes the supplied SQL statement as part of
	// the supplied transaction. Statements are currently executed as the root user.
	ExecuteStatementInTransaction(
		ctx context.Context, opName string, txn *client.Txn, statement string, params ...interface{},
	) (int, error)

	// QueryRowInTransaction executes the supplied SQL statement as part of the
	// supplied transaction and returns the result. Statements are currently
	// executed as the root user.
	QueryRowInTransaction(
		ctx context.Context, opName string, txn *client.Txn, statement string, qargs ...interface{},
	) (tree.Datums, error)

	// QueryRowsInTransaction executes the supplied SQL statement as part of the
	// supplied transaction and returns the resulting rows. Statements are currently
	// executed as the root user.
	QueryRowsInTransaction(
		ctx context.Context, opName string, txn *client.Txn, statement string, qargs ...interface{},
	) ([]tree.Datums, sqlbase.ResultColumns, error)

	QueryRows(
		ctx context.Context, opName string, statement string, qargs ...interface{},
	) ([]tree.Datums, sqlbase.ResultColumns, error)
}
