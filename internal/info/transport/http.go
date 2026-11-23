package transport

import (
	"context"
	"net/http"
	"reflect"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/domain"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/service"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

const BasePath = "/api/v1"

type InfoHandler struct {
	service service.InfoService
}

func NewInfoHandler(infoService service.InfoService) *InfoHandler {
	return &InfoHandler{service: infoService}
}

type infoOutput struct {
	Body   *domain.InfoResponse
	Status int
}

// RegisterRoutes mounts the typed endpoints on the project's existing Gin group.
// A nil service can be used to export the contract without starting the app.
func (h *InfoHandler) RegisterRoutes(router *gin.Engine, group *gin.RouterGroup) *huma.OpenAPI {
	config := huma.DefaultConfig("Template API", "1.0.0")
	config.Info.Description = "Contract for the typed service information endpoint."
	config.Servers = []*huma.Server{{URL: BasePath}}
	// Keep the existing JSON body and let the server decide where to expose docs.
	config.CreateHooks = nil
	config.OpenAPIPath, config.DocsPath, config.SchemasPath = "", "", ""
	api := humagin.NewWithGroup(router, group, config)
	operation := huma.Operation{
		OperationID: "get-service-info",
		Method:      http.MethodGet,
		Path:        "/info",
		Summary:     "Get service information",
		Tags:        []string{"Service"},
	}
	api.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"ServiceJWT": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
	}
	registry := api.OpenAPI().Components.Schemas
	operation.Security = []map[string][]string{{"ServiceJWT": {}}}
	operation.Responses = map[string]*huma.Response{
		"401": {
			Description: "Missing or invalid bearer token",
			Content: map[string]*huma.MediaType{
				"application/json": {
					Schema: &huma.Schema{
						Type:       huma.TypeObject,
						Required:   []string{"error"},
						Properties: map[string]*huma.Schema{"error": {Type: huma.TypeString}},
					},
				},
			},
		},
		"503": {
			Description: "A required dependency is unavailable",
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: registry.Schema(reflect.TypeFor[domain.InfoResponse](), true, "")},
			},
		},
	}

	huma.Register(api, operation, h.getInfo)
	return api.OpenAPI()
}

func (h *InfoHandler) getInfo(ctx context.Context, _ *struct{}) (*infoOutput, error) {
	info := h.service.GetInfo(ctx)
	status := http.StatusOK
	if !info.Dependencies.Database.Reachable && info.Dependencies.Database.Status != "disabled" {
		status = http.StatusServiceUnavailable
	}
	return &infoOutput{Status: status, Body: info}, nil
}
