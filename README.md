# Navigator

Navigator is an easy to use Helm Chart Repository.

Navigator indexes charts directly from a git repository, and is able to archive
and serve all versions of a Chart by reading the commit history. All operations
are done in-memory and chart packages are generated on the fly.

Cloning and indexing https://github.com/kubernetes/charts takes under 10
seconds. Archiving and serving a chart takes milliseconds.

## Usage

### Command line

```
# Example: Offical Helm git repository, stable directory
$ navigator --url https://github.com/kubernetes/charts#stable --interval 5m

# Example: Offical Helm git repository, stable + incubator directory
$ navigator --url https://github.com/kubernetes/charts#stable,incubator --interval 5m

# Example: A bunch of different repositories from GitHub
$ navigator \
	--url https://github.com/KubeLondon/london.k8s.uk#chart \
	--url https://github.com/IBM-Blockchain/ibm-container-service#helm-charts \
	--url https://github.com/ibm-cloud-architecture/charts#stable,incubator
```
