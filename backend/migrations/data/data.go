// Package data embeds the SQL migrations of the travel database in the binary. Each top-level .sql file or
// folder is one migration, in name order; a folder's .sql files run together in name order.
// While v0.1.0 is the only release, change 001_init instead of adding migrations.
package data

import "embed"

//go:embed *
var FS embed.FS
