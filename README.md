# Navigator

Navigator is an easy to use Helm Chart Repository.

Navigator indexes charts directly from a git repository, and is able to archive and serve all versions by reading the commit history. All operations are done in-memory and chart packages are generated on the fly.

Cloning and indexing https://github.com/kubernetes/charts takes under 10 seconds. Archiving and serving a chart takes milliseconds.

##### Example: Offical Helm git repository, stable directory
```
$ docker run saracen/navigator \
	--url https://github.com/kubernetes/charts#stable --interval 5m

level=info event=add-repository repository=https://github.com/kubernetes/charts directories=stable
level=info event=fetching repository=https://github.com/kubernetes/charts took=1.69526324s
level=info event=indexing repository=https://github.com/kubernetes/charts charts=129 versions=1158 took=4.7088991s
level=info event=listening transport=HTTP addr=:8081
```
---
##### Example: Offical Helm git repository, stable + incubator directory
```
$ docker run saracen/navigator \
	--url https://github.com/kubernetes/charts#stable,incubator --interval 5m

level=info event=add-repository repository=https://github.com/kubernetes/charts directories=stable,incubator
level=info event=fetching repository=https://github.com/kubernetes/charts took=1.624256521s
level=info event=indexing repository=https://github.com/kubernetes/charts charts=129 versions=1158 took=5.0983349s
level=info event=listening transport=HTTP addr=:8081
```
---
###### Example: A bunch of different repositories from GitHub
```
$ docker run saracen/navigator \
	--url https://github.com/KubeLondon/london.k8s.uk#chart \
	--url https://github.com/IBM-Blockchain/ibm-container-service#helm-charts \
	--url https://github.com/ibm-cloud-architecture/charts#stable,incubator

level=info event=add-repository repository=https://github.com/KubeLondon/london.k8s.uk directories=chart
level=info event=add-repository repository=https://github.com/IBM-Blockchain/ibm-container-service directories=helm-charts
level=info event=add-repository repository=https://github.com/ibm-cloud-architecture/charts directories=stable,incubator
level=info event=fetching repository=https://github.com/KubeLondon/london.k8s.uk took=1.339515554s
level=info event=indexing repository=https://github.com/KubeLondon/london.k8s.uk charts=1 versions=1 took=2.732451ms
level=info event=fetching repository=https://github.com/IBM-Blockchain/ibm-container-service took=729.217541ms
level=info event=indexing repository=https://github.com/IBM-Blockchain/ibm-container-service charts=5 versions=5 took=37.527198ms
level=info event=fetching repository=https://github.com/ibm-cloud-architecture/charts took=1.049479095s
level=info event=indexing repository=https://github.com/ibm-cloud-architecture/charts charts=8 versions=16 took=10.5084ms
level=info event=listening transport=HTTP addr=:8081
```
