package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestGetClusters_ReturnsSnapshot(t *testing.T) {
	s := newTestServer(t)
	s.Cache().replaceAll([]capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned", Provider: "aws", K8sVersion: "v1.30.2"},
		{Name: "stg-1", Namespace: "staging", Phase: "Pending"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var got struct {
		Clusters []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			Phase     string `json:"phase"`
			Provider  string `json:"provider"`
			Version   string `json:"version"`
		} `json:"clusters"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Clusters, 2)
	assert.Equal(t, "prod-eu-1", got.Clusters[0].Name)
	assert.Equal(t, "prod", got.Clusters[0].Namespace)
	assert.Equal(t, "Provisioned", got.Clusters[0].Phase)
	assert.Equal(t, "aws", got.Clusters[0].Provider)
	assert.Equal(t, "v1.30.2", got.Clusters[0].Version)
}

func TestGetClusters_EmptyIsEmptyArrayNotNull(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"clusters":[]`)
}
