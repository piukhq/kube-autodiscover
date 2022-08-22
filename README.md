# Kubernetes Cluster Autodiscovery

There be an API. There be a poster. There be a cli app.

The poster container post's the cluster name and CA cert to the API every 5mins.

The cli app reconfigures the local kubeconfig basied on the API output.

## Shipping via Jamf

Sorry, this might be some rushed documentation as I doubt this'll need updating very often.

We're using [`gon`](https://github.com/mitchellh/gon) for signing and notarizing this application, basically, you can't run apps on macOS without Apple saying you're allowed to use them. This includes internally developed apps. So, yes, we're technically giving Apple the ability to look at our Kubernetes Clusters with a valid CA cert, but, obviously not giving them auth to do more than that. ANYWAY!

1) Release by hand via GitHub using the usual tagging approach
3) Make a `build` directory in this project folder
2) Download the compiled artifacts from GitHub for macOS for amd64 and arm64 and move them into the `build` directory`
4) Make a file called `gon.json` in the `build` directory with the following content:
```json
{
    "source": ["b2k_darwin_amd64", "b2k_darwin_arm64"],
    "bundle_id": "com.bink.b2k",
    "apple_id": {
        "username": "app@bink.sh",
        "password": "upva-gkmc-ledt-fskm",
	"provider": "HC34M8YE55"
    },
    "sign": {
        "application_identity": "Developer ID Application: Loyalty Angels Ltd (HC34M8YE55)"
    },
    "zip" :{
        "output_path" : "b2k.zip"
    }
}
```
5) Ensure you have the Apple Signing Certificate in your macOS Keychain
    * Found via 1Password secret titled: `Apple Developer ID Application Certificate (HC34M8YE55)`
6) run `gon gon.conf`
7) Hopefully, this all worked and you now have a zip file with your compiled, signed, notarized apps in. Next, upload these blobs to [Azure Blob Storage](https://portal.azure.com/#@bink.com/resource/subscriptions/0add5c8e-50a6-4821-be0f-7a47c879b009/resourceGroups/storage/providers/Microsoft.Storage/storageAccounts/binkpublic/overview)
8) Finally, update the URLs in: [Jamf](https://bink.jamfcloud.com/view/settings/computer-management/scripts/62?tab=script)

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
