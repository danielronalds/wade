package openapi

import _ "embed"

//go:embed swagger.json
var swaggerJSON []byte

// JSON returns the embedded OpenAPI JSON specification.
func JSON() []byte {
	return swaggerJSON
}
