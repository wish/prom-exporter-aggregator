package pkg

import (
	"context"
	"fmt"
	"io"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context/ctxhttp"
)

const (
	metricsFormat = "text/plain; version=0.0.4; charset=utf-8"
)

type Config struct {
	Exporters []Exporter
}

type Exporter struct {
	Name string
	URL  string
}

type Options interface {
	Timeout() time.Duration
	MetricKey() string
}

func parse(buf io.Reader, labelKey, labelVal string) ([]*dto.MetricFamily, error) {
	ft := expfmt.Format(metricsFormat)
	dec := expfmt.NewDecoder(buf, ft)
	out := []*dto.MetricFamily{}

	err := (error)(nil)
	for err == nil {
		f := &dto.MetricFamily{}
		err = dec.Decode(f)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse: %v", err)
		}
		out = append(out, f)
	}

	// TODO(tvi): Catch duplicate keys.
	for _, f := range out {
		for _, m := range f.Metric {
			name, value := labelKey, labelVal[:]
			m.Label = append([]*dto.LabelPair{&dto.LabelPair{
				Name:  &name,
				Value: &value,
			}}, m.Label...)
		}
	}
	return out, nil
}

func Gather(ctx context.Context, config *Config, o Options) ([]*dto.MetricFamily, error) {
	ch := make(chan ftch, len(config.Exporters))
	for _, exporter := range config.Exporters {
		go fetch(ctx, exporter.URL, exporter.Name, ch, o)
	}
	out := []*dto.MetricFamily{}
	for i := 0; i < len(config.Exporters); i++ {
		f := <-ch
		if f.err == nil {
			out = append(out, f.mf...)
		} else {
			logrus.Warnf("A exporter (%v) returned error: %v", f.name, f.err)
		}
	}
	return out, nil
}

type ftch struct {
	mf   []*dto.MetricFamily
	name string
	err  error
}

func fetch(ctx context.Context, url, name string, ch chan ftch, o Options) {
	logrus.Debugf("Fetching %v : %v", name, url)
	reqCtx, cancel := context.WithTimeout(ctx, o.Timeout())
	defer cancel()

	res, err := ctxhttp.Get(reqCtx, nil, url)
	if err != nil {
		logrus.Debugf("HTTP Get failed %v (%v): %v", name, url, err)
		ch <- ftch{[]*dto.MetricFamily{}, name, err}
		return
	}
	defer res.Body.Close()
	mfs, err := parse(res.Body, o.MetricKey(), name)
	if err != nil {
		logrus.Debugf("parse failed %v (%v): %v", name, url, err)
		ch <- ftch{[]*dto.MetricFamily{}, name, err}
		return
	}
	ch <- ftch{mfs, name, nil}
}
