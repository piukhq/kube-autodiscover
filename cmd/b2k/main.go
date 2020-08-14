package main

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"time"
)

type Cluster struct {
	Name     string    `json:"cluster"`
	URL      string    `json:"url"`
	CA       string    `json:"ca"`
	LastSeen time.Time `json:"-"`
}

func main() {
	req, err := http.NewRequest("GET", "https://cluster-autodiscover.uksouth.bink.sh", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to make POST request")
	}

	req.Header.Set("Authorization", "Token aa2d765e-b701-4ed2-8550-60a54af0e38d")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get clusters")
	}

	var clusters []Cluster

	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		log.Fatal().Err(err).Msg("Failed to decode JSON")
	}
	resp.Body.Close()

	fmt.Printf("JSON: %#v\n\n\n", clusters)

	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get kubectl config")
	}
	fmt.Printf("clientcfg: %#v\n\n\n", clientCfg)
}
