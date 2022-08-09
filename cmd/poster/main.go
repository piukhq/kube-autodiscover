package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func main() {
	cluster := os.Getenv("CLUSTER_NAME")
	if cluster == "" {
		log.Fatal().Msg("Missing environment variable CLUSTER_NAME")
	}

	cluster_url := os.Getenv("EXTERNAL_URL")
	if cluster_url == "" {
		log.Fatal().Msg("Missing environment variable EXTERNAL_URL")
	}

	url := os.Getenv("API")
	if url == "" {
		url = "https://cluster-autodiscover.uksouth.bink.sh"
	}
	log.Info().Msgf("Sening cluster info to %s", url)

	ca_cert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get kube CA cert")
	}

	payload := map[string]string{
		"cluster": cluster,
		"url":     cluster_url,
		"ca":      string(ca_cert),
	}

	payloadString, err := json.Marshal(payload)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to convert payload to JSON")
	}

	for {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadString))
		if err != nil {
			log.Error().Err(err).Msg("Failed to make POST request")
			<-time.After(5 * time.Minute)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Token aa2d765e-b701-4ed2-8550-60a54af0e38d")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("Failed to post API request")
			<-time.After(5 * time.Minute)
			continue
		}
		resp.Body.Close()
		log.Info().Msgf("Posted JSON, response status code: %d", resp.StatusCode)

		<-time.After(5 * time.Minute)
	}

}
