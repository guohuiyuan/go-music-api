package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guohuiyuan/music-lib/model"
)

func TestQualityEndpointReturnsStaticCatalog(t *testing.T) {
	recorder := serveJSON("/api/v1/music/qualities?source=netease")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Code int `json:"code"`
		Data struct {
			Source    string                     `json:"source"`
			Qualities []model.AudioQualityOption `json:"qualities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != http.StatusOK || response.Data.Source != "netease" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if len(response.Data.Qualities) == 0 {
		t.Fatal("quality catalog is empty")
	}
}

func TestQualityCompatibilityRoute(t *testing.T) {
	recorder := serveJSON("/music/qualities?source=qq")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"source":"qq"`) {
		t.Fatalf("compatibility response missing qq source: %s", recorder.Body.String())
	}
}

func TestRankingValidationRoutes(t *testing.T) {
	tests := []struct {
		path string
		code int
	}{
		{"/api/v1/ranking/toplist", http.StatusBadRequest},
		{"/api/v1/ranking/hot?source=unknown", http.StatusBadRequest},
		{"/music/toplist_songs", http.StatusBadRequest},
	}
	for _, test := range tests {
		recorder := serveJSON(test.path)
		if recorder.Code != test.code {
			t.Errorf("GET %s status = %d, want %d; body = %s", test.path, recorder.Code, test.code, recorder.Body.String())
		}
	}
}

func serveJSON(path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	SetupRouter().ServeHTTP(recorder, request)
	return recorder
}
