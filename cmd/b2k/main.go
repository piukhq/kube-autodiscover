package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Cluster struct {
	Name     string    `json:"cluster"`
	URL      string    `json:"url"`
	CA       string    `json:"ca"`
	LastSeen time.Time `json:"-"`
}

var (
	version   string
	sha1      string
	buildTime string
)

var CLI struct {
	Email   string `help:"Bink email used for Kubernetes Auth" env:"BINK_KUBE_EMAIL"`
	Version bool   `help:"Display b2k version" short:"V"`
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	kong.Parse(&CLI)

	if CLI.Version {
		fmt.Printf("Version: %s Git SHA: %s Build Time: %s\n", version, sha1, buildTime)
		os.Exit(0)
	}

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

	clusterMap := make(map[string]*Cluster)
	for index, cluster := range clusters {
		clusterMap[cluster.Name] = &clusters[index]
	}

	log.Info().Msgf("Autodiscovered %d clusters", len(clusters))
	// fmt.Printf("JSON: %#v\n\n\n", clusters)

	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get kubectl config")
	}
	changed := false

	// Find auth info
	authInfo := ""
	for name := range clientCfg.AuthInfos {
		if strings.HasSuffix(name, "@bink.com") {
			authInfo = name
		}
	}
	if len(authInfo) == 0 {
		if len(CLI.Email) == 0 {
			log.Fatal().Err(err).Msg("No bink user found, use --email argument to specify")
		}

		newAuthInfo := &api.AuthInfo{
			AuthProvider: &api.AuthProviderConfig{
				Name: "azure",
				Config: map[string]string{
					"client-id":    "aeb43981-c317-4f08-97be-aeed19f91cb1",
					"environment":  "AzurePublicCloud",
					"apiserver-id": "d250be93-618a-45e5-b9cf-6a156f536a00",
					"tenant-id":    "a6e2367a-92ea-4e5a-b565-723830bcc095",
				},
			},
		}
		clientCfg.AuthInfos[CLI.Email] = newAuthInfo
		authInfo = CLI.Email
		log.Info().Msgf("Added %s authInfo", CLI.Email)
		changed = true
	}

	// Remove clusters that
	clustersToRemove := make([]string, 0)
	contextsToRemove := make([]string, 0)

	for k := range clientCfg.Clusters {
		// If we start with a uksouth or ??? prefix, and we're not in the list of autodiscovered clusters
		// remove
		if validCluster(k) {
			if !inClusterSlice(clusters, k) {
				clustersToRemove = append(clustersToRemove, k)
			} else {
				// Cluster exists, so check and fixup the data
				if clientCfg.Clusters[k].Server != clusterMap[k].URL {
					log.Info().Msgf("Updating cluster %s URL to %s", k, clusterMap[k].URL)
					clientCfg.Clusters[k].Server = clusterMap[k].URL
					changed = true
				}

				// Check CA is the same
				clusterCA := []byte(clusterMap[k].CA)
				if !bytes.Equal(clientCfg.Clusters[k].CertificateAuthorityData, clusterCA) {
					log.Info().Msgf("Updating cluster %s CA", k)
					clientCfg.Clusters[k].CertificateAuthorityData = clusterCA
					changed = true
				}
			}
		}
	}
	for k := range clientCfg.Contexts {
		if validCluster(k) && !inClusterSlice(clusters, k) {
			contextsToRemove = append(contextsToRemove, k)
		}
	}
	// Remove
	for _, item := range clustersToRemove {
		log.Info().Msgf("Removing %s cluster", item)
		delete(clientCfg.Clusters, item)
		changed = true
	}
	for _, item := range contextsToRemove {
		log.Info().Msgf("Removing %s context", item)
		delete(clientCfg.Contexts, item)
		changed = true
	}

	// Add clusters & contexts we autodiscovered
	clustersToAdd := make(map[string]*api.Cluster)
	contextsToAdd := make(map[string]*api.Context)

	for _, cluster := range clusters {
		if _, exists := clientCfg.Clusters[cluster.Name]; !exists {
			// Cluster does not exist
			clustersToAdd[cluster.Name] = &api.Cluster{
				Server:                   cluster.URL,
				InsecureSkipTLSVerify:    false,
				CertificateAuthorityData: []byte(cluster.CA),
			}
			log.Info().Msgf("Adding %s cluster", cluster.Name)
		}
		if _, exists := clientCfg.Contexts[cluster.Name]; !exists {
			// Context does not exist
			contextsToAdd[cluster.Name] = &api.Context{
				Cluster:   cluster.Name,
				AuthInfo:  authInfo,
				Namespace: "default",
			}
			log.Info().Msgf("Adding %s context", cluster.Name)
		}
	}
	for key, data := range clustersToAdd {
		clientCfg.Clusters[key] = data
		changed = true
	}
	for key, data := range contextsToAdd {
		clientCfg.Contexts[key] = data
		changed = true
	}

	// Merge current config to disk
	configAccess := clientcmd.NewDefaultPathOptions()
	if err := clientcmd.ModifyConfig(configAccess, *clientCfg, true); err != nil {
		log.Fatal().Err(err).Msg("Failed to update kube config")
	}

	if changed {
		log.Info().Msg("Kubernetes config updated")
	} else {
		log.Info().Msg("Kubernetes config unchanged")
	}
}

func validCluster(clusterName string) bool {
	return strings.HasPrefix(clusterName, "uksouth-")
}

func inClusterSlice(s []Cluster, v string) bool {
	for _, item := range s {
		if item.Name == v {
			return true
		}
	}
	return false
}
