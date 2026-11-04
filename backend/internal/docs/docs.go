package docs

import (
	_ "embed"
	"github.com/swaggo/swag"
)

//go:embed swagger.json
var document string
var SwaggerInfo = &swag.Spec{InfoInstanceName: "swagger", SwaggerTemplate: document, BasePath: "/api/v1", Title: "Video CMS API", Version: "1.0"}

func init() { swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo) }
