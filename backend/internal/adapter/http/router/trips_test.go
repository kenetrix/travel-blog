package router_test

import (
	"encoding/json"
	"strings"
	"testing"
)

type tripBody struct {
	Id         string
	Title      string
	StartsOn   string
	Visibility string
	Countries  []string
	StopCount  int
	Stops      []struct {
		Id      string
		Name    string
		Country string
		Arrival string
	}
}

func decodeTrip(t *testing.T, body []byte) tripBody {
	t.Helper()
	var trip tripBody
	if err := json.Unmarshal(body, &trip); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
	return trip
}

func TestTripFlow(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")

	rec := do(t, h, "GET", "/api/trips", "", me)
	etag := rec.Header().Get("ETag")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "[]" || etag == "" {
		t.Fatalf("empty trips: %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "POST", "/api/trips", `{"title":"Danube","startsOn":"2024-05-01","endsOn":"","stops":[
		{"name":"Vienna","country":"AUT","lat":48.2,"lng":16.37,"arrival":"2024-05-01"},
		{"name":"Budapest","country":"HUN","lat":47.5,"lng":19.04}]}`, me)
	if rec.Code != 201 || rec.Header().Get("ETag") == etag {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	trip := decodeTrip(t, rec.Body.Bytes())
	if trip.Visibility != "private" || trip.StartsOn != "2024-05-01" || strings.Join(trip.Countries, ",") != "AUT,HUN" || len(trip.Stops) != 2 || trip.Stops[0].Arrival != "2024-05-01" {
		t.Errorf("created = %+v", trip)
	}

	rec = do(t, h, "GET", "/api/trips/"+trip.Id, "", me)
	if rec.Code != 200 || decodeTrip(t, rec.Body.Bytes()).Title != "Danube" {
		t.Errorf("get: %d %s", rec.Code, rec.Body)
	}

	// Keep Budapest first, drop Vienna, add Bratislava
	rec = do(t, h, "PUT", "/api/trips/"+trip.Id, `{"title":"Danube","stops":[
		{"id":"`+trip.Stops[1].Id+`","name":"Budapest","country":"HUN","lat":47.5,"lng":19.04},
		{"name":"Bratislava","country":"SVK","lat":48.14,"lng":17.1}]}`, me)
	if rec.Code != 200 {
		t.Fatalf("replace: %d %s", rec.Code, rec.Body)
	}
	if got := decodeTrip(t, rec.Body.Bytes()); got.Stops[0].Id != trip.Stops[1].Id || strings.Join(got.Countries, ",") != "HUN,SVK" || got.StartsOn != "" {
		t.Errorf("replaced = %+v", got)
	}

	rec = do(t, h, "GET", "/api/trips", "", me)
	var list []tripBody
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].StopCount != 2 || len(list[0].Stops) != 0 {
		t.Errorf("list = %s", rec.Body)
	}

	// Another user neither sees nor changes the trip
	other := register(t, h, "noah")
	for _, req := range []struct{ method, body string }{
		{"GET", ""},
		{"PUT", `{"title":"Mine","stops":[]}`},
		{"DELETE", ""},
	} {
		rec := do(t, h, req.method, "/api/trips/"+trip.Id, req.body, other)
		if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"code":"trip_not_found"`) {
			t.Errorf("%s by another user: %d %s", req.method, rec.Code, rec.Body)
		}
	}

	if rec := do(t, h, "DELETE", "/api/trips/"+trip.Id, "", me); rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/api/trips/"+trip.Id, "", me); rec.Code != 404 {
		t.Errorf("deleted trip: %d", rec.Code)
	}
}

func TestTripProblems(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")
	cases := map[string]string{
		"title_required":      `{"title":" ","stops":[]}`,
		"invalid_date_range":  `{"title":"T","startsOn":"2024-05-14","endsOn":"2024-05-01","stops":[]}`,
		"name_required":       `{"title":"T","stops":[{"name":"","country":"AUT","lat":1,"lng":1}]}`,
		"invalid_coordinates": `{"title":"T","stops":[{"name":"X","country":"AUT","lat":1,"lng":181}]}`,
		"invalid_code":        `{"title":"T","stops":[{"name":"X","country":"at","lat":1,"lng":1}]}`,
		"unknown_stop":        `{"title":"T","stops":[{"id":"6f1d1f0e-8c1a-4a4e-9a52-2d4c1f1c0b11","name":"X","country":"AUT","lat":1,"lng":1}]}`,
	}
	for code, body := range cases {
		rec := do(t, h, "POST", "/api/trips", body, me)
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), `"code":"`+code+`"`) {
			t.Errorf("%s: %d %s", code, rec.Code, rec.Body)
		}
	}
	if rec := do(t, h, "GET", "/api/trips", "", nil); rec.Code != 401 {
		t.Errorf("trips without login: %d", rec.Code)
	}
}
