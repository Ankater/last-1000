package testsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync/atomic"
)

type Config struct {
	Exec  func(ctx context.Context, query string, args []any) (driver.Result, error)
	Query func(ctx context.Context, query string, args []any) (driver.Rows, error)
	Ping  func(ctx context.Context) error
}

func Open(cfg Config) (*sql.DB, error) {
	name := fmt.Sprintf("testsql-%d", atomic.AddUint64(&driverID, 1))
	sql.Register(name, &testDriver{cfg: cfg})
	return sql.Open(name, "")
}

func Rows(columns []string, values ...[]any) driver.Rows {
	return &testRows{
		columns: columns,
		values:  values,
	}
}

var driverID uint64

type testDriver struct {
	cfg Config
}

func (d *testDriver) Open(name string) (driver.Conn, error) {
	return &testConn{cfg: d.cfg}, nil
}

type testConn struct {
	cfg Config
}

func (c *testConn) Prepare(query string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare not supported in test driver")
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("transactions not supported in test driver")
}

func (c *testConn) Ping(ctx context.Context) error {
	if c.cfg.Ping == nil {
		return nil
	}
	return c.cfg.Ping(ctx)
}

func (c *testConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.cfg.Exec == nil {
		return nil, fmt.Errorf("unexpected ExecContext call: %s", query)
	}
	return c.cfg.Exec(ctx, query, namedValues(args))
}

func (c *testConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.cfg.Query == nil {
		return nil, fmt.Errorf("unexpected QueryContext call: %s", query)
	}
	return c.cfg.Query(ctx, query, namedValues(args))
}

func (c *testConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

type testRows struct {
	columns []string
	values  [][]any
	index   int
}

func (r *testRows) Columns() []string {
	return r.columns
}

func (r *testRows) Close() error {
	return nil
}

func (r *testRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}

	row := r.values[r.index]
	for i := range row {
		dest[i] = row[i]
	}
	r.index++
	return nil
}

func namedValues(values []driver.NamedValue) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value.Value)
	}
	return args
}
