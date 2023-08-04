# kubedmp
## Intruction
The `kubectl cluster-info dump` command dumps cluster information for debugging and diagnosing cluster problems.
The output consists of a number of json documents and container logs. It can be very large if all of these are dumped into a single file.
kubedmp parses the dump file(s) and displays the output nicely in a simliar way as kubectl command's output.

## Usage

The use of kubedmp is similar to kubectl; it has several sub commands. By default it reads file `./cluster-info.dump` as input; a different file can be specified with flag `-f path/to/dump/file`; if the dump is a directory, specify it with `-d path/to/dump/dir`.

```
Available Commands:
  describe    Show details of a specific resource
  get         Display one or many resources
  logs        Print the logs for a container in a pod
  show        show all objects in cluster info dump file in ps output format

Flags:
  -d, --dumpdir string    Path to dump dir
  -f, --dumpfile string   Path to dump file (default "./cluster-info.dump")
  -h, --help              help for kubedmp
  -v, --version           get version
```

* kubedmp get
```
Display one or many resources of a type, which can be node/no, pod/po, service/svc, deployment/deploy, daemonset/ds, replicaset/rs or event. 
Prints a table of the most important information about resources of the specific type.

Usage:
  kubedmp get TYPE [-n NAMESPACE | -A]

Examples:
  # Lists all pods in kube-system namespace in ps output format, the output contains all fields in 'kubectl get -o wide'
  kubedmp get po -n kube-system
  
  # List all nodes
  kubedmp get no
```
* kubedmp describe
```
Show details of a specific resource. Print a detailed description of the selected resource.
It can only show detais of one resource, whose type is either node/no, pod/po, service/svc, deployment/deploy, daemonset/ds or replicaset/rs.

Usage:
  kubedmp describe TYPE RESOURCE_NAME [-n NAMESPACE]

Examples:
  # Describe a node
  $ kubedmp describe no juju-ceba75-k8s-2
  
  # Describe a pod in kube-system namespace
  $ kubedmp describe po coredns-6bcf44f4cc-j9wkq -n kube-system
```
* kubedmp logs
```
Print the logs for a container in a pod or specified resource.
If the pod has more than one container, and a container name is not specified, logs of all containers will be printed out.

Usage:
  kubedmp logs POD_NAME [-n NAMESPACE] [-c CONTAINER_NAME]

Examples:
  # Return logs from pod nginx with all containers
  kubedmp logs nginx
  
  # Return logs of ruby container logs from pod web-1
  kubectl logs web-1 -c ruby
```
* kubedmp show
```
show all objects in cluster info dump file in ps output format

Usage:
  kubedmp show [flags]
```
## Installation

### Tarball
Download tarball from [releases](https://github.com/shundezhang/kubedmp/releases) and extract it.

### Homebrew
Homebrew supports MacOS and Linux.
```
brew tap shundezhang/kubedmp
brew install shundezhang/kubedmp/kubedmp
```
### Snap
Snap supports Linux.
```
snap install kubedmp
```

## Development

Run from source
```
go run cmd/main.go -f ~/cluster-info.dump get po
```

Build binary
```
make bin
```
Or use goreleaser
```
goreleaser --snapshot --skip-publish --clean
```

Release
```
git tag version-number
git push --tags
```
