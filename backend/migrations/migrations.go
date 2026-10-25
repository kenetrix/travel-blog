// Package migrations embeds the SQL migrations in the binary. Files run in name order;
// only append new migrations, never change existing ones.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
