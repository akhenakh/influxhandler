package influxhandler

import (
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
	// config is a Config struct to configure the BufferedHandler behaviour
	config Config
	// series stores the Series before flushing
	series []*client.Series
	// ch is a channel fed with new series
	ch chan (*client.Series)
	// lastF the last time data went flushed
	lastF time.Time
	// mutex to protect the series, a FIFO channel is not safe enough
	// as the series may be dequeued by the timeout from another goroutine
	// TODO: could it be avoided by sending nil ch from the timeout ?
	sync.Mutex
}

// NewHandler create a new Handler that will log every http request into InfluxDB
// Each request generate a flush to the DB
func NewHandler(name string, c *client.Client) *Middleware {
	return NewBufferedHandler(name, c, Config{})
}

// NewBufferedHandler create a new Handler that will log every http request into InfluxDB
// Series will be flushed according to config options
func NewBufferedHandler(name string, c *client.Client, config Config) *Middleware {
	m := &Middleware{name: name,
		client: c,
		config: config,
		ch:     make(chan *client.Series),
		series: make([]*client.Series, config.MaxSeriesCount),
		lastF:  time.Now()}

	go func() {
		scopy := make([]*client.Series, m.config.MaxSeriesCount)
		timeout := make(chan bool, 1)

		for {

			if m.config.MaxDuration != 0 {
				go func() {
					time.Sleep(m.config.MaxDuration)
					timeout <- true
				}()
			}

			select {
			case s := <-m.ch:
				m.Lock()

				m.series = append(m.series, s)

				if len(m.series) >= m.config.MaxSeriesCount && m.config.MaxDuration != 0 && time.Since(m.lastF) < m.config.MaxDuration {
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
				m.Unlock()

			case <-timeout:
				// check if we need to flush
				m.Lock()

				if len(m.series) > 0 && time.Since(m.lastF) > m.config.MaxDuration {
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
				m.Unlock()
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
