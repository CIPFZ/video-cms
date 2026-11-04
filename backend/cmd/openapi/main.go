// Generate the route-level contract from the same router used by the server.
package main

import (
	"encoding/json"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/server"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"os"
	"reflect"
	"strings"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	sc := svc.NewServiceContext()
	sc.Config = &config.Config{System: config.System{RouterPrefix: "/api/v1"}, Observable: config.Observability{Exporter: "none"}}
	sc.Logger = zap.NewNop()
	router := server.InitRouters(sc)
	paths := map[string]any{}
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") || strings.Contains(route.Path, "/swagger/") {
			continue
		}
		path := strings.TrimPrefix(route.Path, "/api/v1")
		op := map[string]any{"summary": route.Method + " " + path, "produces": []string{"application/json"}, "responses": map[string]any{"200": map[string]any{"description": "CMS response; code 0 means success", "schema": map[string]any{"$ref": "#/definitions/Response"}}}}
		if body, ok := requests[path]; ok {
			op["consumes"] = []string{"application/json"}
			op["parameters"] = []any{map[string]any{"in": "body", "name": "body", "schema": requestSchema(reflect.TypeOf(body))}}
		}

		if strings.HasPrefix(path, "/open/") {
			op["security"] = []any{map[string]any{"ApiTokenAuth": []string{}}}
		}
		if strings.HasPrefix(path, "/sys/") {
			op["security"] = []any{map[string]any{"ApiKeyAuth": []string{}}}
		}
		methods, ok := paths[path].(map[string]any)
		if !ok {
			methods = map[string]any{}
			paths[path] = methods
		}
		methods[strings.ToLower(route.Method)] = op
	}
	spec := map[string]any{"swagger": "2.0", "info": map[string]any{"title": "Video CMS API", "version": "1.0", "description": "Routes generated from the backend router; request fields generated from system DTOs. Responses use the CMS envelope."}, "basePath": "/api/v1", "paths": paths, "securityDefinitions": map[string]any{"ApiTokenAuth": map[string]any{"type": "apiKey", "name": "X-API-Token", "in": "header"}, "ApiKeyAuth": map[string]any{"type": "apiKey", "name": "x-token", "in": "header"}}, "definitions": map[string]any{"Response": map[string]any{"type": "object", "required": []string{"code"}, "properties": map[string]any{"code": map[string]any{"type": "integer"}, "msg": map[string]any{"type": "string"}, "data": map[string]any{}}}}}
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("internal/docs/swagger.json", append(data, '\n'), 0644); err != nil {
		panic(err)
	}
}
