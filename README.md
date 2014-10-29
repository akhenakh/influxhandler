influxhandler
=============

A generic net/http Go InfluxDB handler

Use it in place of your logger:

```golang
// initialize influxdb
conf := &influxdb.ClientConfig{
	Host:     "myhostname:8086",
	Username: "xxx",
	Password: "oxoxo",
	Database: "test",
}

client, err := influxdb.NewClient(conf)
if err != nil {
	log.Fatal(err)
}
h := influxhandler.NewHandler("myhostname.resp_time", client)
handler = h.Handler(demoHandler)
http.ListenAndServe(":8080", handler)
```

Then query for `code` as status code, `duration`, `url` and `method`

```sql
SELECT duration FROM resp_time WHERE code = 200
```

Note: NewHandler is using the REST api on every requests which is probably not what you want, on heavy traffic prefer NewBufferedHandler which will queue the series then flush them.

```golang
h := influxhandler.NewBufferedHandler("myhostname.resp_time", client, influxhandler.Config{MaxSeriesCount: 1000, MaxDuration: 1 * time.Minute})
handler = h.Handler(demoHandler)
http.ListenAndServe(":8080", handler)
```

![demo](https://github.com/akhenakh/martini-influxdb/raw/master/img/graph.png)

Status: not finished yet