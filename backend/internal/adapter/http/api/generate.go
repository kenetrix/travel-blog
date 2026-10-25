// Package api holds the server interface and models generated from api/openapi.yaml.
package api

// oapi-codegen cannot follow references into other files, so the split
// contract is first bundled into a single, gitignored file.
//go:generate node ../../../../../node_modules/@redocly/cli/bin/cli.js bundle ../../../../../api/openapi.yaml -o openapi.bundle.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 -config ../../../../oapi-codegen.yaml openapi.bundle.yaml
