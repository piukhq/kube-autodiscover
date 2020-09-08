# Kubernetes Cluster Autodiscovery

There be an API. There be a poster. There be a cli app.

The poster container post's the cluster name and CA cert to the API every 5mins.

The cli app reconfigures the local kubeconfig basied on the API output.

## Layout

* cmd/api - contains the simple web api
* cmd/b2k - contains the cli app
* cmd/poster - contains the posting container

The repo makes use of the Gitlab feature where it'll only run stages based on changes so if you edit the API it won't rebuild the poster or the cli app.

When the cli app is built, the result will be attached to the CI job as an artifact.

## API

GET https://cluster-autodiscover.uksouth.bink.sh

Will return a list of maps containing cluster name, external url and CA

E.g.
```
[
  {
    "cluster": "prod0",
    "url": "https://prod0.uksouth.bink.sh:4000",
    "ca": "---- blah"
  }
]
```

---

POST https://cluster-autodiscover.uksouth.bink.sh

Will accept a map of cluster, url and CA to add to the internal collection of clusters

E.g.
```
{
  "cluster": "prod0",
  "url": "https://prod0.uksouth.bink.sh:4000",
  "ca": "---- blah"
}
```
