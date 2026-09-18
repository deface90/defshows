// Package openapi holds defShows service contracts and their code generation
// directives. Run `make generate` (or `go generate ./...`) to regenerate.
package openapi

//go:generate go tool oapi-codegen -config cfg/auth.server.yaml auth.yaml
//go:generate go tool oapi-codegen -config cfg/shows.server.yaml shows.yaml
//go:generate go tool oapi-codegen -config cfg/tracking.server.yaml tracking.yaml
//go:generate go tool oapi-codegen -config cfg/notifications.server.yaml notifications.yaml
//go:generate go tool oapi-codegen -config cfg/notes.server.yaml notes.yaml
//go:generate go tool oapi-codegen -config cfg/admin.server.yaml admin.yaml
//go:generate go tool oapi-codegen -config cfg/users.server.yaml users.yaml
//go:generate go tool oapi-codegen -config ../../deps/tmdb/openapi/cfg/tmdb.client.yaml ../../deps/tmdb/openapi/tmdb.yaml
//go:generate go tool oapi-codegen -config ../../deps/omdb/openapi/cfg/omdb.client.yaml ../../deps/omdb/openapi/omdb.yaml
