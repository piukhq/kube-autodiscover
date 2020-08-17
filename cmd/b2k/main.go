package main

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"net/http"
	"strings"
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

	log.Info().Msgf("Autodiscovered %d clusters", len(clusters))
	// fmt.Printf("JSON: %#v\n\n\n", clusters)

	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get kubectl config")
	}

	// Find auth info
	authInfo := ""
	for name := range clientCfg.AuthInfos {
		if strings.HasSuffix(name, "@bink.com") {
			authInfo = name
		}
	}
	if len(authInfo) == 0 {
		log.Fatal().Msg("Could not find authInfo aka user, todo need to be able to add this...")
	}
	// kubectl config set-credentials yourusername@bink.com \
	// --auth-provider=azure \
	// --auth-provider-arg=environment=AzurePublicCloud \
	// --auth-provider-arg=client-id=aeb43981-c317-4f08-97be-aeed19f91cb1 \
	// --auth-provider-arg=apiserver-id=d250be93-618a-45e5-b9cf-6a156f536a00 \
	// --auth-provider-arg=tenant-id=a6e2367a-92ea-4e5a-b565-723830bcc095

	changed := false

	// Remove clusters that
	clustersToRemove := make([]string, 0)
	contextsToRemove := make([]string, 0)

	for k := range clientCfg.Clusters {
		// If we start with a uksouth or ??? prefix, and we're not in the list of autodiscovered clusters
		// remove
		if validCluster(k) && !inClusterSlice(clusters, k) {
			clustersToRemove = append(clustersToRemove, k)
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
