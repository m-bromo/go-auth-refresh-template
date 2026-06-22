package docs

import _ "embed"

// OpenAPISpec is the embedded Swagger/OpenAPI contract served by the API.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
