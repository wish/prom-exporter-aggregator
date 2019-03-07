package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ghodss/yaml"
	flags "github.com/jessevdk/go-flags"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"

	"github.com/wish/prom-aggregator/pkg"
)

const (
	contentTypeHeader     = "Content-Type"
	contentEncodingHeader = "Content-Encoding"
	acceptEncodingHeader  = "Accept-Encoding"
)

type ops struct {
	LogLevel string `long:"log-level" env:"LOG_LEVEL" description:"Log level" default:"info"`
	Config   string `long:"config" short:"f" description:"Config file path" required:"yes"`
	BindAddr string `long:"bind-address" short:"p" env:"BIND_ADDRESS" default:":9560" description:"address for binding metrics listener"`

	Time  time.Duration `long:"timeout" short:"t" default:"1s" description:"Timeout for exporter fetch"`
	Label string        `long:"label-key" short:"l" default:"agent" description:"Metric label name for exporters"`
}

func (o *ops) Timeout() time.Duration { return o.Time }
func (o *ops) MetricKey() string      { return o.Label }

var (
	config *pkg.Config
	opts   *ops
)

func main() {
	opts = &ops{}
	parser := flags.NewParser(opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		// If the error was from the parser, then we can simply return
		// as Parse() prints the error already
		if _, ok := err.(*flags.Error); ok {
			os.Exit(1)
		}
		logrus.Fatalf("Error parsing flags: %v", err)
	}

	// Use log level
	level, err := logrus.ParseLevel(opts.LogLevel)
	if err != nil {
		logrus.Fatalf("Unknown log level %s: %v", opts.LogLevel, err)
	}
	logrus.SetLevel(level)

	// Set the log format to have a reasonable timestamp
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logrus.SetFormatter(formatter)

	b, err := ioutil.ReadFile(filepath.Clean(opts.Config))
	if err != nil {
		panic(err)
	}
	c := &pkg.Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		panic(err)
	}
	dedup := map[string]bool{}
	for _, b := range c.Exporters {
		if _, ok := dedup[b.Name]; ok {
			panic(fmt.Errorf("Name: %v is not unique", b.Name))
		}
		dedup[b.Name] = true
	}
	config = c
	http.HandleFunc("/", handler)
	http.HandleFunc("/healthcheck", handler)
	http.HandleFunc("/metrics", metricsHandler)
	logrus.Infof("Started listening on %v", opts.BindAddr)
	logrus.Fatal(http.ListenAndServe(opts.BindAddr, nil))
}

func metricsHandler(rsp http.ResponseWriter, req *http.Request) {
	mfs, err := pkg.Gather(req.Context(), config, opts)
	if err != nil {
		httpError(rsp, err)
		return
	}

	sort.Sort(pkg.MergeFamilies(mfs))
	out := []*dto.MetricFamily{}
	for _, m := range mfs {
		sort.Sort(pkg.Metrics(m.Metric))
		if *m.Name != "" {
			out = append(out, m)
		}
	}

	contentType := expfmt.Negotiate(req.Header)
	header := rsp.Header()
	header.Set(contentTypeHeader, string(contentType))

	w := io.Writer(rsp)
	enc := expfmt.NewEncoder(w, contentType)

	var lastErr error
	for _, mf := range out {
		if err := enc.Encode(mf); err != nil {
			lastErr = err
			httpError(rsp, err)
			return
		}
	}

	if lastErr != nil {
		httpError(rsp, lastErr)
	}
}

func httpError(rsp http.ResponseWriter, err error) {
	rsp.Header().Del(contentEncodingHeader)
	http.Error(
		rsp,
		"An error has occurred while serving metrics:\n\n"+err.Error(),
		http.StatusInternalServerError,
	)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK\n")
}
