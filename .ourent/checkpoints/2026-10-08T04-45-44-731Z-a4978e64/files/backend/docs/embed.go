// Package docs embeds the OpenAPI specification and a Swagger UI page.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte

const SwaggerHTML = `<!doctype html><html><head><meta charset="utf-8"><title>API docs</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="ui"></div><script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({url:'/api/v1/openapi.yaml',dom_id:'#ui'})</script></body></html>`
