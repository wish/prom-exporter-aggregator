package pkg

import (
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"
)

type MergeFamilies []*dto.MetricFamily

func (a MergeFamilies) Len() int      { return len(a) }
func (a MergeFamilies) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a MergeFamilies) Less(i, j int) bool {
	return strings.Compare(*a[i].Name, *a[j].Name) == -1
}

func concatLabel(lps []*dto.LabelPair) string {
	out := ""
	for _, lp := range lps {
		out += fmt.Sprintf("%v%v", *lp.Name, *lp.Value)
	}
	return out
}

type Metrics []*dto.Metric

func (a Metrics) Len() int      { return len(a) }
func (a Metrics) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a Metrics) Less(i, j int) bool {
	return strings.Compare(
		concatLabel(a[i].Label),
		concatLabel(a[j].Label)) == -1
}
