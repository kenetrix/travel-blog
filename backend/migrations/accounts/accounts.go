// Package accounts embeds the SQL migrations of the accounts database, which holds
// the users apart from the travel data, so deleting that keeps them (same rules as
// package data).
package accounts

import "embed"

//go:embed *
var FS embed.FS
