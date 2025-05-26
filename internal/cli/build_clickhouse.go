//go:build clickhouse

package cli

import (
	_ "github.com/ClickHouse/clickhouse-go"
	_ "github.com/abramad-labs/histomigrate/database/clickhouse"
)
