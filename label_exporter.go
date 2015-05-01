package main

import (
	"errors"
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const METRIC_LINE_RE = `^([a-zA-Z_][a-zA-Z0-9_:]+)(\{[^{}]+\})? ([^ ]+)( [^ ]+)?$`

var (
	// Metrics
	processed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "label_exporter",
			Subsystem: "requests",
			Name:      "total",
			Help:      "The number of localhost:port/path requests served.",
		},
		[]string{"code", "port"},
	)
	unprocessed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "label_exporter",
		Subsystem: "metrics_unprocessed",
		Name:      "total",
		Help:      "The number of metrics unable to be processed.",
	})

	// Command line flags
	listenAddress = flag.String("web.listen-address", ":9900", "Address to listen on")
	proxyHost     = flag.String("proxy-host", "localhost", "Host to proxy requests against")
	labelsDir     = flag.String("labels-dir", "/tmp/target", "Directory to find *.label in")
	flagset       = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
)

func fetchMetricsEndpoint(url string) ([]byte, error) {
	var metrics []byte
	resp, err := http.Get(url)
	if err != nil {
		return metrics, err
	} else {
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}
}

func getNewLabels(labels string, overrides map[string]string) string {
	before := make(map[string]string)
	if len(labels) > 0 {
		before = labelsToMap(labels)
	}
	after := updateMap(before, overrides)
	if len(after) == 0 {
		return ""
	}
	return labelsToString(after)
}

func getOverrides(r *http.Request) map[string]string {
	overrides := make(map[string]string)
	updateOverrides := func(path string, f os.FileInfo, err error) error {
		re, _ := regexp.Compile(`^.+?([^/]+)\.label$`)
		match := re.FindStringSubmatch(path)
		if len(match) > 0 {
			value, err := ioutil.ReadFile(path)
			if err == nil {
				log.Println("Loaded override from:", path)
				overrides[match[1]] = strings.Trim(string(value), "\n")
			} else {
				log.Println("Unable to read:", path, err)
			}
		}
		return nil
	}
	filepath.Walk(*labelsDir, updateOverrides)
	return updateMap(urlValuesToMap(r.URL.Query()), overrides)
}

func labelInjectingHandler(w http.ResponseWriter, r *http.Request, port string, path string) {
	metrics, err := fetchMetricsEndpoint("http://" + *proxyHost + ":" + port + path)
	if err != nil {
		http.Error(w, "# "+err.Error(), http.StatusServiceUnavailable)
		processed.WithLabelValues(strconv.Itoa(http.StatusServiceUnavailable), port).Inc()
	} else {
		lines := strings.Split(string(metrics), "\n")
		re, _ := regexp.Compile(METRIC_LINE_RE)
		overrides := getOverrides(r)
		for idx, line := range lines {
			io.WriteString(w, processLine(line, idx, re, overrides)+"\n")
		}
		processed.WithLabelValues("200", port).Inc()
	}
}

func labelsToMap(labels string) map[string]string {
	_labels := labels[1 : len(labels)-1]
	m := make(map[string]string)
	for _, label := range strings.Split(_labels, ",") {
		p := strings.Split(label, "=")
		m[p[0]] = p[1][1 : len(p[1])-1]
	}
	return m
}

func labelsToString(m map[string]string) string {
	var keys []string
	var pairs []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pairs = append(pairs, k+`="`+m[k]+`"`)
	}
	new_labels := strings.Join(pairs, ",")
	return "{" + new_labels + "}"
}

func processLine(line string, idx int, re *regexp.Regexp, overrides map[string]string) string {
	if len(line) == 0 || strings.HasPrefix(line, "#") {
		return line
	} else {
		match := re.FindStringSubmatch(line)
		if len(match) == 5 {
			return rewriteLabels(match, overrides)
		} else {
			log.Printf("Unable to process line %v [length=%v]: %v\n",
				idx+1, len(line), line)
			unprocessed.Inc()
			return line
		}
	}
}

func rewriteLabels(match []string, overrides map[string]string) string {
	name := string(match[1])
	labels := string(match[2])
	value := string(match[3])
	timestamp := string(match[4])
	labels = getNewLabels(labels, overrides)
	return name + labels + " " + value + timestamp
}

func getPortPath(path string) (string, string, error) {
	re, _ := regexp.Compile(`^([0-9]+)(/.*)?$`)
	match := re.FindStringSubmatch(path)
	if len(match) > 0 {
		return match[1], match[2], nil
	} else {
		return "", "", errors.New("Regex parsing of path failed")
	}
}

func router(w http.ResponseWriter, r *http.Request) {
	port, path, err := getPortPath(r.URL.Path[1:])
	if err == nil {
		labelInjectingHandler(w, r, port, path)
	} else {
		http.NotFound(w, r)
	}
}

func updateMap(m map[string]string, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m[k] = v
	}
	return m
}

func urlValuesToMap(v url.Values) map[string]string {
	m := make(map[string]string)
	for k, v := range v {
		m[k] = v[0]
	}
	return m
}

func init() {
	prometheus.MustRegister(processed)
	prometheus.MustRegister(unprocessed)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", router)
	http.Handle("/metrics", prometheus.Handler())
	log.Println("Listening on", *listenAddress)
	log.Printf("My metrics: http://%v/metrics\n", *listenAddress)
	log.Printf("Proxied metrics: http://%v/<port>/metrics\n", *listenAddress)
	log.Println("Proxying to:", *proxyHost)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
