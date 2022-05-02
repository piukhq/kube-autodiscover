package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

type Cluster struct {
	Name     string    `json:"cluster"`
	URL      string    `json:"url"`
	CA       string    `json:"ca"`
	LastSeen time.Time `json:"-"`
}

var ClusterMap = make(map[string]Cluster)
var SyncMutex = sync.RWMutex{}

const ClusterTimeout = 5 * time.Minute

func livez(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
}

func handler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "GET" {
		w.WriteHeader(405)
		return
	}

	if req.Header.Get("Authorization") != "Token aa2d765e-b701-4ed2-8550-60a54af0e38d" {
		w.WriteHeader(401)
		return
	}

	if req.Method == "POST" {
		var c Cluster
		if err := json.NewDecoder(req.Body).Decode(&c); err != nil {
			log.Error().Err(err).Msg("Failed to decode JSON response")
			w.WriteHeader(500)
		} else {
			c.LastSeen = time.Now()
			SyncMutex.Lock()
			ClusterMap[c.Name] = c
			SyncMutex.Unlock()
		}

	} else if req.Method == "GET" {
		resp := make([]Cluster, 0)
		now := time.Now()
		toDelete := make([]string, 0)

		SyncMutex.RLock()
		// Loop through map and find clusters that have checked in recently
		// and find old clusters for cleanup
		for key, value := range ClusterMap {
			if value.LastSeen.Add(ClusterTimeout).Before(now) {
				toDelete = append(toDelete, key)
			} else {
				resp = append(resp, value)
			}
		}
		SyncMutex.RUnlock()

		SyncMutex.Lock()
		// Clean up old clusters
		for _, deleteKey := range toDelete {
			delete(ClusterMap, deleteKey)
		}
		SyncMutex.Unlock()

		respBytes, err := json.Marshal(resp)
		if err != nil {
			log.Error().Err(err).Msg("Failed to encode JSON response")
			w.WriteHeader(500)
		} else {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(respBytes)
			if err != nil {
				log.Error().Err(err).Msg("Failed to encode JSON response")
				w.WriteHeader(500)
			}
		}
	}
}

func main() {
	// Setup http logger
	httpLogger := zerolog.New(os.Stdout).With().Timestamp().Str("type", "http").Logger()
	c := alice.New()
	c = c.Append(hlog.NewHandler(httpLogger))
	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	c = c.Append(hlog.RemoteAddrHandler("ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	http.HandleFunc("/livez", livez)
	http.HandleFunc("/readyz", livez)
	http.Handle("/", c.Then(http.HandlerFunc(handler)))

	log.Info().Msg("Serving on :9000")
	err := http.ListenAndServe(":9000", nil)
	log.Error().Err(err).Msg("Caught error whilst serving http")
}
