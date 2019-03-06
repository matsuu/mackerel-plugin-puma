package mppuma

import (
	"flag"
	"log"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// PumaPlugin mackerel plugin for Puma
type PumaPlugin struct {
	Prefix string
	Host   string
	Port   string
	Sock   string
	Token  string
	Single bool
	WithGC bool
}

func merge(m1, m2 map[string]float64) map[string]float64 {
	ans := make(map[string]float64)

	for k, v := range m1 {
		ans[k] = v
	}
	for k, v := range m2 {
		ans[k] = v
	}
	return (ans)
}

// FetchMetrics interface for mackerelplugin
func (p PumaPlugin) FetchMetrics() (map[string]float64, error) {
	ret := make(map[string]float64)

	stats, err := p.getStatsAPI()
	if err != nil {
		return nil, err
	}

	ret = p.fetchStatsMetrics(stats)

	if p.WithGC == false {
		return ret, nil
	}

	gcStats, err := p.getGCStatsAPI()
	if err != nil {
		return nil, err
	}

	gcStatsMetrics, _ := p.fetchGCStatsMetrics(gcStats)

	ret = merge(ret, gcStatsMetrics)

	return ret, nil

}

// GraphDefinition interface for mackerelplugin
func (p PumaPlugin) GraphDefinition() map[string]mp.Graphs {
	graphdef := graphdefStats

	if p.Single == true {
		graphdef = graphdefStatsSingle
	}

	if p.WithGC == false {
		return graphdef
	}

	for k, v := range graphdefGC {
		graphdef[k] = v
	}
	return graphdef
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p PumaPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "puma"
	}
	return p.Prefix
}

// Do the plugin
func Do() {
	var (
		optPrefix   = flag.String("metric-key-prefix", "puma", "Metric key prefix")
		optHost     = flag.String("host", "127.0.0.1", "The bind url to use for the control server")
		optPort     = flag.String("port", "9293", "The bind port to use for the control server")
		optSock     = flag.String("sock", "", "The bind socket to use for the control server")
		optState    = flag.String("state", "", "The bind state file to use for the control server")
		optToken    = flag.String("token", "", "The token to use as authentication for the control server")
		optSingle   = flag.Bool("single", false, "Puma in single mode")
		optWithGC   = flag.Bool("with-gc", false, "Output include GC stats for Puma 3.10.0~")
		optTempfile = flag.String("tempfile", "", "Temp file name")
	)
	flag.Parse()

	var puma PumaPlugin
	puma.Prefix = *optPrefix
	puma.Host = *optHost
	puma.Port = *optPort
	puma.Sock = *optSock
	puma.Token = *optToken
	puma.Single = *optSingle
	puma.WithGC = *optWithGC

	if *optState != "" {
		type State struct {
			Url   string `yaml:"control_url"`
			Token string `yaml:"control_auth_token"`
		}
		file, err := os.Open(*optState)
		if err != nil {
			log.Fatalf("Failed to open %s: %v\n", *optState, err)
		}
		defer file.Close()

		var state State
		err = yaml.NewDecoder(file).Decode(&state)
		if err != nil {
			log.Fatalf("Failed to decode %s as yaml: %v\n", *optState, err)
		}
		u, err := url.Parse(state.Url)
		if err != nil {
			log.Fatalf("Failed to parse %s as url: %v\n", state.Url, err)
		}
		switch u.Scheme {
		case "unix":
			puma.Sock = u.Path
		case "tcp":
			puma.Host = u.Hostname()
			puma.Port = u.Port()
		default:
			log.Fatalf("Unknown scheme: %s\n", u.Scheme)
		}
		puma.Token = state.Token
	}

	helper := mp.NewMackerelPlugin(puma)
	helper.Tempfile = *optTempfile
	helper.Run()
}
