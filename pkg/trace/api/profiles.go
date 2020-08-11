package api

import (
	"fmt"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/logutil"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	// profilingURLTemplate specifies the template for obtaining the profiling URL along with the site.
	profilingURLTemplate = "https://intake.profile.%s/v1/input"
	// profilingURLDefault specifies the default intake API URL.
	profilingURLDefault = "https://intake.profile.datadoghq.com/v1/input"
)

// profilingEndpoint returns the profiling intake API URL based on agent configuration.
func profilingEndpoints() []string {
	if v := config.Datadog.GetString("apm_config.profiling_dd_url"); v != "" {
		return strings.Split(v, ",")
	}
	if site := config.Datadog.GetString("site"); site != "" {
		return []string{fmt.Sprintf(profilingURLTemplate, site)}
	}
	return []string{profilingURLDefault}
}

// profileProxyHandler returns a new HTTP handler which will proxy requests to the profiling intake.
// If the URL can not be computed because of a malformed 'site' config, the returned handler will always
// return http.StatusInternalServerError along with a clarification.
func (r *HTTPReceiver) profileProxyHandler() http.Handler {
	targets := profilingEndpoints()
	proxies := []*httputil.ReverseProxy{}
	for _, target := range targets {
		u, err := url.Parse(target)
		if err != nil {
			log.Errorf("Error parsing intake URL %s: %v", target, err)
			continue
		}
		tags := fmt.Sprintf("host:%s,default_env:%s", r.conf.Hostname, r.conf.DefaultEnv)
		proxy := newProfileProxy(r.conf.NewHTTPTransport(), u, r.conf.APIKey(), tags)
		if proxy != nil {
			proxies = append(proxies, proxy)
		}
	}
	switch len(proxies) {
	case 0:
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			msg := fmt.Sprintf("Profile forwarder is OFF because of invalid intake URL configuration: %v", targets)
			http.Error(w, msg, http.StatusInternalServerError)
		})
	case 1:
		return proxies[0]
	default:
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			proxies[0].ServeHTTP(w, req)
			for _, proxy := range proxies[1:] {
				// for additional endpoints we ignore the response
				proxy.ServeHTTP(&dummyResponseWriter{}, req)
			}
		})
	}
}

// newProfileProxy creates a single-host reverse proxy with the given target, attaching
// the specified apiKey.
func newProfileProxy(transport http.RoundTripper, target *url.URL, apiKey, tags string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL = target
		req.Host = target.Host
		req.Header.Set("DD-API-KEY", apiKey)
		req.Header.Set("Via", fmt.Sprintf("trace-agent %s", info.Version))
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to the default value
			// that net/http gives it: Go-http-client/1.1
			// See https://codereview.appspot.com/7532043
			req.Header.Set("User-Agent", "")
		}
		containerID := req.Header.Get(headerContainerID)
		if ctags := getContainerTags(containerID); ctags != "" {
			req.Header.Set("X-Datadog-Container-Tags", ctags)
		}
		req.Header.Set("X-Datadog-Additional-Tags", tags)
		metrics.Count("datadog.trace_agent.profile", 1, nil, 1)
	}
	logger := logutil.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	return &httputil.ReverseProxy{
		Director:  director,
		ErrorLog:  stdlog.New(logger, "profiling.Proxy: ", 0),
		Transport: transport,
	}
}

type dummyResponseWriter struct{}

func (d *dummyResponseWriter) Header() http.Header {
	return make(map[string][]string)
}

func (d *dummyResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (d *dummyResponseWriter) WriteHeader(statusCode int) {}
