package main

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type Cluster struct {
	Name     string    `json:"cluster"`
	URL      string    `json:"url"`
	CA       string    `json:"ca"`
	LastSeen time.Time `json:"-"`
}

var ClusterMap = make(map[string]Cluster)

const ClusterTimeout = 5 * time.Minute

func livez(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
}

func handler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		var c Cluster
		if err := json.NewDecoder(req.Body).Decode(&c); err != nil {
			log.Error().Err(err).Msg("Failed to decode JSON response")
			w.WriteHeader(500)
		} else {
			c.LastSeen = time.Now()
			ClusterMap[c.Name] = c
		}

	} else if req.Method == "GET" {
		resp := make([]Cluster, 0)
		now := time.Now()
		toDelete := make([]string, 0)

		for key, value := range ClusterMap {
			if value.LastSeen.Add(ClusterTimeout).Before(now) {
				toDelete = append(toDelete, key)
			} else {
				resp = append(resp, value)
			}
		}

		for _, deleteKey := range toDelete {
			delete(ClusterMap, deleteKey)
		}

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
	} else {
		w.WriteHeader(405)
	}
}

func main() {
	http.HandleFunc("/livez", livez)
	http.HandleFunc("/readyz", livez)
	http.HandleFunc("/", handler)

	log.Info().Msg("Serving on :9000")
	err := http.ListenAndServe(":9000", nil)
	log.Error().Err(err).Msg("Caught error whilst serving http")
}
