// Package configmigrations embeds the SQL migrations of the configuration database, which
// holds the settings of the installation apart from the travel data (same rules as
// package migrations).
package configmigrations

import "embed"

//go:embed 001_init
var FS embed.FS
