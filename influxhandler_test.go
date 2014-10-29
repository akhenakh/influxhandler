package influxhandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/influxdb/influxdb/client"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test"))
})

func TestNoClientHandler(t *testing.T) {
	m := New("test.resp_time", nil, Options{})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost/", nil)

	m.Handler(testHandler).ServeHTTP(res, req)
}

func TestHandler(t *testing.T) {
	conf := &client.ClientConfig{
		Host:     "localhost:8086",
		Username: "root",
		Password: "root",
		Database: "test",
	}
	client, err := client.NewClient(conf)
	if err != nil {
		t.Errorf("Can't connect to influxdb", err)
	}

	m := New("test.resp_time", client, Options{})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost/", nil)

	m.Handler(testHandler).ServeHTTP(res, req)
}
