package cli

import (
	"os"
	"testing"
)

// Set KUBEDMP_BENCH_DUMP to a cluster-info dump file to run these benchmarks.
var benchDump = os.Getenv("KUBEDMP_BENCH_DUMP")

func BenchmarkReadFileNoop(b *testing.B) {
	if benchDump == "" {
		b.Skip("set KUBEDMP_BENCH_DUMP to a dump file")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		readFile(benchDump, func([]byte) bool { return true })
	}
}

func BenchmarkGetPods(b *testing.B) {
	if benchDump == "" {
		b.Skip("set KUBEDMP_BENCH_DUMP to a dump file")
	}
	resKind = "Pod"
	resNamespace = "default"
	allNamespaces = true
	resName = ""
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		displayItems = make([]interface{}, 0)
		readFile(benchDump, processDoc)
	}
}

func BenchmarkDescribePod(b *testing.B) {
	if benchDump == "" {
		b.Skip("set KUBEDMP_BENCH_DUMP to a dump file")
	}
	resType = "po"
	resKind = "Pod"
	resNamespace = "cert-manager"
	resName = "cert-manager-7778d64785-mrjv8"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		readFile(benchDump, describeObject)
	}
}
