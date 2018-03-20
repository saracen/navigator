# Navigator

[![Go Report Card](https://goreportcard.com/badge/github.com/saracen/navigator)](https://goreportcard.com/report/github.com/saracen/navigator)
[![GoDoc](https://godoc.org/github.com/saracen/navigation?status.svg)](https://godoc.org/github.com/saracen/navigator)
[![codecov](https://codecov.io/gh/saracen/navigator/branch/master/graph/badge.svg)](https://codecov.io/gh/saracen/navigator)

Navigator is an easy to use Helm Chart Repository written in Go.

Navigator indexes charts directly from a git repository, and is able to archive and serve all versions by reading the commit history. All operations are done in-memory and chart packages are generated on the fly.

Cloning and indexing https://github.com/kubernetes/charts takes under 10 seconds. Archiving and serving a chart takes milliseconds.

##### Required
- A git repository

##### *Not* Required
- A cloud storage backend and the associated configuration
- CI/CD processes to build and upload charts

## Installation
### Binaries
Binaries can be found on the [releases](https://github.com/saracen/navigator/releases) page.

### Docker
```
docker pull saracen/navigator
```

## Usage
```
Usage of navigator:
  -http-addr string
        HTTP listen address (default ":8080")
  -interval duration
        Poll interval for git repository updates (default 5m0s)
  -url value
        Git repository to index
```

The fragment part of the Git URL can be used to specify the directories (separated by a comma) that you want indexed.

For example, `https://github.com/<username>/<repo>.git#my-charts,my-test-charts/dev` will index the directories `my-charts` and `my-test-charts`. These will go to the chart index `default`, available at `http://localhost:8081/default/index.yaml`.

You can specify different chart indexes by using the format `<directory>@<index>`. For example, `#my-charts@stable,my-test-charts/dev@dev` will make charts under `my-charts` be available at `http://localhost:8081/stable/index.yaml` and `my-test-charts` be available at `http://localhost:8081/dev/index.yaml`.

Chart indexes are required if your repository uses a [dependency alias](https://github.com/kubernetes/helm/blob/master/docs/charts.md#alias-field-in-requirementsyaml) as the alias will resolve to an index of the same name.

## Examples
##### Example: Mirror of official Helm git repository, stable directory
```
$ docker run saracen/navigator \
	--url https://github.com/kubernetes/charts#stable@stable --interval 5m

level=info event=add-repository repository=https://github.com/kubernetes/charts directories=stable@stable
level=info event=fetching repository=https://github.com/kubernetes/charts took=1.7683338s
level=info event=indexing repository=https://github.com/kubernetes/charts head=c0e593513184490f895b3a82375934545677c309 took=4.7310006s
level=info event=listening transport=HTTP addr=:8080

# add to helm
$ helm repo add official-mirror-stable http://localhost:8080/stable
```
---
##### Example: Mirror of official Helm git repository, stable + incubator directory
```
$ docker run saracen/navigator \
	--url https://github.com/kubernetes/charts#stable@stable,incubator@incubator --interval 5m

level=info event=add-repository repository=https://github.com/kubernetes/charts directories=stable@stable,incubator@incubator
level=info event=fetching repository=https://github.com/kubernetes/charts took=1.6392525s
level=info event=indexing repository=https://github.com/kubernetes/charts head=c0e593513184490f895b3a82375934545677c309 took=5.970003s
level=info event=listening transport=HTTP addr=:8080

# add to helm
$ helm repo add official-mirror-stable http://localhost:8080/stable
$ helm repo add official-mirror-incubator http://localhost:8080/incubator
```
---
##### Example: A bunch of different repositories from GitHub

__Note__: Only aggregate repositories that you trust. Navigator will add the most recently committed chart and version to an index.
This could allow a bad repository to override a chart version from another repository. You can choose to mitigate this issue by
ensuring that each repository has it's own index.
```
$ docker run saracen/navigator \
	--url https://github.com/KubeLondon/london.k8s.uk#chart@kubelondon \
	--url https://github.com/IBM-Blockchain/ibm-container-service#helm-charts@ibm-container-service \
	--url https://github.com/ibm-cloud-architecture/charts#stable@ibm-stable,incubator@ibm-incubator

level=info event=add-repository repository=https://github.com/KubeLondon/london.k8s.uk directories=chart@kubelondon
level=info event=add-repository repository=https://github.com/IBM-Blockchain/ibm-container-service directories=helm-charts@ibm-container-service
level=info event=add-repository repository=https://github.com/ibm-cloud-architecture/charts directories=stable@ibm-stable,incubator@ibm-incubator
level=info event=fetching repository=https://github.com/KubeLondon/london.k8s.uk took=2.1232248s
level=info event=indexing repository=https://github.com/KubeLondon/london.k8s.uk head=8f1e5e796e57c0f18462dd091ee28322e16ace16 took=1.972ms
level=info event=fetching repository=https://github.com/IBM-Blockchain/ibm-container-service took=1.0524576s
level=info event=indexing repository=https://github.com/IBM-Blockchain/ibm-container-service head=7a98ae518d2f0441d6fb5a82c7617630b70f295c took=24.002ms
level=info event=fetching repository=https://github.com/ibm-cloud-architecture/charts took=1.499308s
level=info event=indexing repository=https://github.com/ibm-cloud-architecture/charts head=d2848c03956a57e0ef07a0791f26c9baa0739cde took=22.0083ms
level=info event=listening transport=HTTP addr=:8080

# add to helm
$ helm repo add kubelondon http://localhost:8080/kubelondon
$ helm repo add ibm-container-service http://localhost:8080/ibm-container-service
$ helm repo add ibm-stable http://localhost:8080/ibm-stable
$ helm repo add ibm-incubator http://localhost:8080/ibm-incubator
```