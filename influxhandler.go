package influxhandler

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/influxdb/influxdb/client"
)

type Config struct {
	// MaxSeries the maximum series kept before flushing them to InfluxDB
	MaxSeriesCount int
	// MaxDuration is the max duration we keep series before flushing them to InfluxDB
	MaxDuration time.Duration
}

type Middleware struct {
	// Name is the name used to store the serie
	name string
	// Client is the InfluxDB connection client
	client *client.Client
	// Options
	config Config
	// lock protects the series
	lock sync.Mutex
	// series stores the Series before flushing
	series []*client.Series
	// c is a channel fed with new series
	ch chan (*client.Series)
	// t is the last time we have flushed data
	t time.Time
}

// NewHandler create a new Handler that will log every http request into InfluxDB
// Each request generate a flush to the DB
func NewHandler(name string, c *client.Client) *Middleware {
	return NewBuferedHandler(name, c, Config{})
}

// NewBuferedHandler create a new Handler that will log every http request into InfluxDB
// Series will be flushed according to config options
func NewBuferedHandler(name string, c *client.Client, config Config) *Middleware {
	m := &Middleware{name: name,
		client: c,
		config: config,
		lock:   sync.Mutex{},
		ch:     make(chan *client.Series),
		series: make([]*client.Series, config.MaxSeriesCount),
		t:      time.Now()}

	var timeout chan bool
	if m.config.MaxDuration != 0 {
		timeout = make(chan bool, 1)
		go func() {
			time.Sleep(m.config.MaxDuration)
			timeout <- true
		}()
	}

	select {
	case <-m.ch:
		// a read from ch has occurred
	case <-time.After(m.config.MaxDuration):
		// check if we need to flush

	}

	return m
}

// writeSeries saves or bufferizes the series
func (m *Middleware) WriteSeries(s *client.Series) {
	if m.config.MaxDuration == 0 && m.config.MaxSeriesCount == 0 {
		err := m.client.WriteSeries([]*client.Series{s})
		if err != nil {
			log.Println("influxhandler", err)
		}
		return
	}
}

// Handler logs the request as it goes in and the response as it goes out
func (m *Middleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m.client == nil {
			h.ServeHTTP(w, req)
			return
		}
		start := time.Now()
		h.ServeHTTP(w, req)
		fmt.Println(w.Header())
		t := time.Since(start)
		s := &client.Series{
			Name:    "resp_time",
			Columns: []string{"duration", "code", "url", "method"},
			Points: [][]interface{}{
				// TODO: get the real status code
				[]interface{}{int64(t / time.Millisecond), 200, req.RequestURI, req.Method},
			},
		}
		m.WriteSeries(s)
	})
}
