/*
MIT License

Copyright (c) 2026 gounix

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package gometricsvr

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

type (
	// data for in the prometheus header
	StatT struct {
		MetricName  string
		Description string
		MetricType  string
	}
	MetricLineT struct {
		MetricName string
		Labels     map[string]string
		Value      float64
	}
	MetricsT struct {
		mu     sync.Mutex
		Header []StatT
		Lines  []MetricLineT
	}
)

var metrics MetricsT

func PutHeader(metricName string, description string, metricType string) {
	var line StatT

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	line.MetricName = metricName
	line.Description = description
	line.MetricType = metricType
	metrics.Header = append(metrics.Header, line)
}

func compareMap(one map[string]string, two map[string]string) bool {
	for key, value := range one {
		if two[key] != value {
			return false
		}
	}
	return true
}

func PutLine(metricName string, value float64, labels map[string]string) {

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	slog.Info("gometricsvr.PutLine", "value", value, "labels", labels)
	for index, existing := range metrics.Lines {
		if existing.MetricName == metricName && compareMap(existing.Labels, labels) {
			metrics.Lines[index].Value = value
			return
		}
	}

	// no existing entry found, create a new one
	var newLine MetricLineT

	newLine.MetricName = metricName
	newLine.Labels = labels
	newLine.Value = value

	metrics.Lines = append(metrics.Lines, newLine)
}

func dumpHeader(w http.ResponseWriter) {

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	slog.Info("gometricsvr.dumpHeader")
	for _, stat := range metrics.Header {
		fmt.Fprintf(w, "# HELP %s %s\n", stat.MetricName, stat.Description)
		fmt.Fprintf(w, "# TYPE %s %s\n", stat.MetricName, stat.MetricType)
	}
}

func dumpLines(w http.ResponseWriter) {
	var sb strings.Builder

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	for _, entry := range metrics.Lines {
		sb.WriteString(fmt.Sprintf("%s(", entry.MetricName))
		index := 0
		for key, value := range entry.Labels {
			if index != 0 {
				sb.WriteString(", ")
			}
			index = index + 1
			sb.WriteString(fmt.Sprintf("%s=\"%s\"", key, value))
		}
		sb.WriteString(fmt.Sprintf(") %f\n", entry.Value))
	}
	fmt.Fprintf(w, sb)
	slog.Info("gometricsvr.dumpLines", "line", sb)
}

func logRequest(r *http.Request) {
	slog.Info("frontend.logRequest", "Host", r.Host, "Method", r.Method, "Url", r.URL.Path, "UserAgent", r.UserAgent())
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {

	logRequest(r)
	dumpHeader(w)
	dumpLines(w)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	//if data.Alive(1) {
	fmt.Fprintf(w, "OK")
	//} else {
	//w.WriteHeader(http.StatusNotFound)
	//}
}

func server(port int) {
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)

	addr := fmt.Sprintf(":%d", port)

	slog.Info("frontend.Server", "listen", addr)
	http.ListenAndServe(addr, nil)
}

func StartServer(port int) {
	go server(port)
}
