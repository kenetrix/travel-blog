package legal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRoutes(t *testing.T) {
	env := map[string]string{
		"LEGAL_NAME":    " Max Mustermann ",
		"LEGAL_ADDRESS": "Musterstr. 1 | 12345 Musterstadt |",
		"LEGAL_EMAIL":   "max@example.com",
	}
	mux := http.NewServeMux()
	Routes(func(k string) string { return env[k] })(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/public/legal", nil))

	var got notice
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := notice{Name: "Max Mustermann", Address: []string{"Musterstr. 1", "12345 Musterstadt"}, Email: "max@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestUnsetIsEmptyNotNull(t *testing.T) {
	mux := http.NewServeMux()
	Routes(func(string) string { return "" })(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/public/legal", nil))
	if body := rec.Body.String(); body != "{\"name\":\"\",\"address\":[],\"email\":\"\"}\n" {
		t.Errorf("body = %q", body)
	}
}
