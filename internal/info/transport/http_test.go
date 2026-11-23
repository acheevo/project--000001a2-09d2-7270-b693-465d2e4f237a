package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubInfoService struct {
	info *domain.InfoResponse
}

func (s stubInfoService) GetInfo(context.Context) *domain.InfoResponse {
	return s.info
}

func TestTypedInfoKeepsDependencyFailureResponse(t *testing.T) {
	info := &domain.InfoResponse{
		Dependencies: domain.DependencyStatus{Database: domain.DatabaseStatus{Status: "unhealthy"}},
	}
	router := gin.New()
	contract := NewInfoHandler(stubInfoService{info}).RegisterRoutes(router, router.Group(BasePath))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, BasePath+"/info", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	body, err := json.Marshal(info)
	require.NoError(t, err)
	require.JSONEq(t, string(body), response.Body.String())
	require.Equal(t, "bearer", contract.Components.SecuritySchemes["ServiceJWT"].Scheme)
	require.Contains(t, contract.Paths["/info"].Get.Responses, "503")
}
