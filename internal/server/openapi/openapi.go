package openapi

import _ "embed"

//go:embed swagger.json
var swaggerJSON []byte

func JSON() []byte {
	return swaggerJSON
}
