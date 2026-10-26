## version 1.0.0

`logs` supports `deployment/NAME`, `daemonset/NAME` and `statefulset/NAME` to print the logs of all selected pods.
When a pod has multiple containers and `-c` is not specified, `logs` now prints every container's logs, including for sosreport dumps. sosreport logs use the same `==== START/END logs for container ... ====` headers as `logs.txt`.
`describe` now includes an Events section populated from events in the dump.
`get` and `describe` accept `TYPE/NAME` as shorthand for `TYPE NAME`.

## version 0.9.0

Supports cluster-info collected by sosreport 4.7.1+
