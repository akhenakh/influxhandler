package influxhandler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/influxdb/influxdb/client"
)

var demoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test"))
})

func TestNoClientHandler(t *testing.T) {
	h := NewHandler("test.resp_time", nil)
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost/", nil)

	h.Handler(demoHandler).ServeHTTP(res, req)
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

	h := NewHandler("test.resp_time", client)
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost/", nil)

	h.Handler(demoHandler).ServeHTTP(res, req)
}

func TestHandlerWithMaxTime(t *testing.T) {
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

	h := NewBufferedHandler("test.resp_time", client, Config{MaxDuration: 50 * time.Millisecond})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost/", nil)

	time.Sleep(100 * time.Millisecond)

	h.Handler(demoHandler).ServeHTTP(res, req)

	time.Sleep(100 * time.Millisecond)

}
