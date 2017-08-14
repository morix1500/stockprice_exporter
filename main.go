package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const namespace = "stockprice_exporter"

var (
	listenAddress = flag.String(
		"web.listen-address",
		":9010",
		"Address on which to expose metrics and web interface.",
	)
	metricsPath = flag.String(
		"web.telemetry-path",
		"/metrics",
		"Path under which to expose metrics.",
	)
	tickerSymbol = flag.String(
		"ticker-symbol",
		"0",
		"An identification code used to identify a publicly traded corporation on a particular stock market.",
	)
	stockExchangeCode = flag.String(
		"stock-exchange-code",
		"TYO",
		"https://www.google.com/intl/en/googlefinance/disclaimer/?ei=QpGRWejgDNOG0ATRtIiQCA",
	)
)

type Exporter struct {
	close  prometheus.Gauge
	open   prometheus.Gauge
	high   prometheus.Gauge
	low    prometheus.Gauge
	volume prometheus.Gauge
}

type CsvRecord struct {
	date   string
	close  float64
	open   float64
	high   float64
	low    float64
	volume float64
}

func newExporter() *Exporter {
	return &Exporter{
		close: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "close",
			Help:      "close",
		}),
		open: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "open",
			Help:      "open",
		}),
		high: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "high",
			Help:      "high",
		}),
		low: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "low",
			Help:      "low",
		}),
		volume: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "volume",
			Help:      "volume",
		}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.close.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.StockPrice(ch)
}

func (e *Exporter) StockPrice(ch chan<- prometheus.Metric) {
	api_url := "https://www.google.com/finance/getprices?q=%s&x=%s&i=300&p=1m&f=d,c,v,o,h,l&df=cpct&auto=1&ei=4rrIWJHoIYya0QS1i4IQ"
	res, err := http.Get(fmt.Sprintf(api_url,
		*tickerSymbol,
		*stockExchangeCode))

	if err != nil {
		log.Fatal(err.Error())
	}
	defer res.Body.Close()

	sc := bufio.NewScanner(res.Body)
	for i := 1; sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			log.Fatal(err.Error())
			break
		}

		if i < 8 {
			continue
		}

		if strings.HasPrefix(sc.Text(), "TIMEZONE_OFFSET") {
			continue
		}

		csv_record, err := parseCsv(sc.Text())
		if err != nil {
			log.Fatal(err.Error())
		}

		e.close.Set(csv_record.close)
		e.close.Collect(ch)

		e.open.Set(csv_record.open)
		e.open.Collect(ch)

		e.high.Set(csv_record.high)
		e.high.Collect(ch)

		e.low.Set(csv_record.low)
		e.low.Collect(ch)

		e.volume.Set(csv_record.volume)
		e.volume.Collect(ch)

		break
	}
}

func parseCsv(line string) (CsvRecord, error) {
	line_array := strings.Split(line, ",")

	csv_record := CsvRecord{}
	csv_record.date = line_array[0]

	var err error

	csv_record.close, err = strconv.ParseFloat(line_array[1], 64)
	if err != nil {
		log.Fatal(err.Error())
		return csv_record, err
	}

	csv_record.high, err = strconv.ParseFloat(line_array[2], 64)
	if err != nil {
		log.Fatal(err.Error())
		return csv_record, err
	}

	csv_record.low, err = strconv.ParseFloat(line_array[3], 64)
	if err != nil {
		log.Fatal(err.Error())
		return csv_record, err
	}

	csv_record.open, err = strconv.ParseFloat(line_array[4], 64)
	if err != nil {
		log.Fatal(err.Error())
		return csv_record, err
	}

	csv_record.volume, err = strconv.ParseFloat(line_array[5], 64)
	if err != nil {
		log.Fatal(err.Error())
		return csv_record, err
	}

	return csv_record, nil
}

func main() {
	flag.Parse()

	exporter := newExporter()
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
                        <head><title>StockPrice Exporter</title></head>
                        <body>
                        <h1>StockPrice Exporter</h1>
                        <p><a href="` + *metricsPath + `">Metrics</a></p>
                        </body>
                        </html>`))
	})
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
