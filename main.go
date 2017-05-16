package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	dto "github.com/prometheus/client_model/go"
)

var (
	log = logrus.New()

	// Metrics
	proxyCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "label_exporter",
			Subsystem: "proxied_request_count",
			Name:      "total",
			Help:      "The number of localhost:port/path requests served.",
		},
		[]string{"port"},
	)
	errCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "label_exporter",
			Subsystem: "errors",
			Name:      "total",
			Help:      "The number of errors.",
		},
		[]string{"type"},
	)

	// Command line flags
	listenAddress = flag.String("web.listen-address", ":9900", "Address to listen on")
	acceptPrefix  = flag.String("accept.prefix", "", "Accept header prefix to be used")
	proxyHost     = flag.String("proxy-host", "localhost", "Host to proxy requests against")
	labelsDir     = flag.String("labels-dir", "/tmp/target", "Directory to find *.label in")
	flagset       = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
)

type labelMap map[string]string

func (lm labelMap) Update(m labelMap) labelMap {
	for k, v := range m {
		lm[k] = v
	}
	return lm
}

func (lm labelMap) ToLabelPairs() []*dto.LabelPair {
	pairs := []*dto.LabelPair{}
	for n, v := range lm {
		var name, value string
		name = n
		value = v
		pairs = append(pairs, &dto.LabelPair{
			Name:  &name,
			Value: &value,
		})
	}
	return pairs
}

func labelMapFromLabelPair(pairs []*dto.LabelPair) labelMap {
	lm := labelMap{}
	for _, p := range pairs {
		lm[*p.Name] = *p.Value
	}
	return lm
}

func getOverrides(r *http.Request) labelMap {
	overrides := labelMap{}
	labelGlob := filepath.Join(*labelsDir, "/*.label")
	files, err := filepath.Glob(labelGlob)
	if err != nil {
		errCounter.WithLabelValues("list-labels-dir").Inc()
		log.WithFields(logrus.Fields{"glob": labelGlob, "err": err}).Error("Unble list labelsdir")
	}
	if len(files) == 0 {
		log.WithFields(logrus.Fields{"glob": labelGlob}).Info("No label files found")
	}
	for _, path := range files {
		value, err := ioutil.ReadFile(path)
		if err == nil {
			overrides[strings.TrimSuffix(filepath.Base(path), ".label")] = strings.Trim(string(value), "\n")
		} else {
			errCounter.WithLabelValues("read-label-file").Inc()
			log.WithFields(logrus.Fields{"path": path, "err": err}).Error("Unable to read")
		}
	}
	return updateMap(urlValuesToMap(r.URL.Query()), overrides)
}

func relabel(r *http.Request, port string, path string) (map[string]*dto.MetricFamily, error) {
	overrides := getOverrides(r)
	req, err := http.NewRequest("GET", "http://"+*proxyHost+":"+port+path, nil)
	if err != nil {
		errCounter.WithLabelValues("http-request-create").Inc()
		return nil, errors.Wrap(err, "Failed to create http request")
	}
	req.Header.Set("Accept", *acceptPrefix+r.Header.Get("Accept"))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errCounter.WithLabelValues("http-get").Inc()
		return nil, errors.Wrap(err, "Failed to fetch downstream metrics")
	}
	defer resp.Body.Close()
	parser := expfmt.TextParser{}
	parsed, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		errCounter.WithLabelValues("metric-parsing").Inc()
		return nil, errors.Wrap(err, "Failed to parse downstream metrics")
	}
	for _, samples := range parsed {
		for _, metrics := range samples.Metric {
			metrics.Label = labelMapFromLabelPair(metrics.Label).Update(overrides).ToLabelPairs()
		}
	}
	proxyCounter.WithLabelValues(port).Inc()
	return parsed, nil
}

func getPortPath(path string) (string, string, error) {
	re, _ := regexp.Compile(`^([0-9]+)(/.*)?$`)
	match := re.FindStringSubmatch(path)
	if len(match) > 0 {
		return match[1], match[2], nil
	}
	return "", "", errors.New("Regex parsing of path failed")
}

func proxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Via", "https://github.com/TheClimateCorporation/label_exporter")
	port, path, err := getPortPath(r.URL.Path[1:])
	if err != nil {
		errCounter.WithLabelValues("get-port").Inc()
		http.NotFound(w, r)
		return
	}
	parsed, err := relabel(r, port, path)
	if err != nil {
		log.WithFields(logrus.Fields{
			"port": port,
			"path": path,
			"err":  err}).Error("Proxy failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, samples := range parsed {
		expfmt.MetricFamilyToText(w, samples)
	}
}

func updateMap(m labelMap, m2 labelMap) labelMap {
	for k, v := range m2 {
		m[k] = v
	}
	return m
}

func urlValuesToMap(v url.Values) labelMap {
	m := labelMap{}
	for k, v := range v {
		m[k] = v[0]
	}
	return m
}

func init() {
	prometheus.MustRegister(errCounter)
	prometheus.MustRegister(proxyCounter)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", proxy)
	http.Handle("/metrics", prometheus.Handler())
	log.Println("Listening on", *listenAddress)
	log.Println("Looking for labels in:", *labelsDir)
	log.Printf("My metrics: http://%v/metrics", *listenAddress)
	log.Printf("Proxied metrics: http://%v/<port>/metrics", *listenAddress)
	log.Println("Proxying to:", *proxyHost)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
