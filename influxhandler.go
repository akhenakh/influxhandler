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
	// series stores the Series before flushing
	series []*client.Series
	// c is a channel fed with new series
	ch chan (*client.Series)
	// t is the last time we have flushed data
	lastF time.Time
	// protects the series
	sync.Mutex
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
		ch:     make(chan *client.Series),
		series: make([]*client.Series, config.MaxSeriesCount),
		lastF:  time.Now()}

	go func() {
		scopy := make([]*client.Series, m.config.MaxSeriesCount)
		var timeout chan bool

		if m.config.MaxDuration != 0 {
			go func() {
				time.Sleep(m.config.MaxDuration)
				timeout <- true
			}()
		}

		for {
			select {
			case s := <-m.ch:

				m.Lock()
				defer m.Unlock()
				m.series = append(m.series, s)
				if len(m.series) >= m.config.MaxSeriesCount {
					copy(scopy, m.series)
					go func() {
						err := m.client.WriteSeries(scopy)
						if err != nil {
							log.Println("influxhandler", err)
						}
					}()
					m.lastF = time.Now()
					m.series = make([]*client.Series, config.MaxSeriesCount)
					copy(scopy, m.series)
				}

			case <-timeout:
				// check if we need to flush
				m.Lock()
				defer m.Unlock()
				if len(m.series) > 0 && time.Since(m.lastF) > m.config.MaxDuration {
					fmt.Println("flush MaxDuration")
					err := m.client.WriteSeries(m.series)
					if err != nil {
						log.Println("influxhandler", err)
					}
					m.lastF = time.Now()
					m.series = make([]*client.Series, config.MaxSeriesCount)
				}
			}
		}

	}()

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
	m.ch <- s
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
