// Package config embeds the SQL migrations of the configuration database, which
// holds the settings and the user accounts of the installation apart from the travel
// data (same rules as package data).
package config

import "embed"

//go:embed *
var FS embed.FS
