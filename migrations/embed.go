// Package migrations embeds the SQL migration files so the binary needs no disk assets.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
