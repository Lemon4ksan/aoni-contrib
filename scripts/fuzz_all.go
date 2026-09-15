package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type fuzzTarget struct {
	pkg  string
	name string
}

var targets = []fuzzTarget{
	{"./grpc", "FuzzGRPCWebFraming"},
	{"./grpc", "FuzzGRPCStatusParsing"},
	{"./grpc", "FuzzGRPCTimeoutFormat"},
	{"./otel", "FuzzParseTraceParent"},
	{"./otel", "FuzzParseTraceID"},
	{"./otel", "FuzzParseSpanID"},
	{"./otel", "FuzzOTLPSpanAttributesSerialization"},
	{"./otel", "FuzzCarrierPropagation"},
	{"./socket", "FuzzLengthPrefixedFramer"},
	{"./webpush", "FuzzVAPIDKeys"},
	{"./webpush", "FuzzWebPushDecrypt"},
}

func main() {
	fuzzDuration := flag.String("fuzztime", "5s", "duration to fuzz each target")

	flag.Parse()

	fmt.Printf("=== Starting Heavy Fuzzing Suite (%d targets, %s each) ===\n\n", len(targets), *fuzzDuration)

	var failed []string

	startTotal := time.Now()

	for i, tgt := range targets {
		fmt.Printf("[%2d/%2d] Fuzzing %s :: %s (fuzztime=%s) ... ", i+1, len(targets), tgt.pkg, tgt.name, *fuzzDuration)

		start := time.Now()

		var dir string

		pkg := tgt.pkg

		// #nosec G204
		cmd := exec.CommandContext(
			context.Background(),
			"go", "test",
			"-fuzz=^"+tgt.name+"$",
			"-fuzztime="+*fuzzDuration,
			pkg,
		)
		if dir != "" {
			cmd.Dir = dir
		}

		var outBuf bytes.Buffer

		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf

		err := cmd.Run()
		elapsed := time.Since(start).Round(time.Millisecond)

		if err != nil {
			fmt.Printf("FAILED (%s)\n", elapsed)
			fmt.Println("----------------- OUTPUT -----------------")
			fmt.Println(strings.TrimSpace(outBuf.String()))
			fmt.Println("------------------------------------------")

			failed = append(failed, fmt.Sprintf("%s :: %s", tgt.pkg, tgt.name))
		} else {
			fmt.Printf("PASSED (%s)\n", elapsed)
		}
	}

	totalElapsed := time.Since(startTotal).Round(time.Second)
	fmt.Printf("\n=== Fuzzing Suite Completed in %s ===\n", totalElapsed)

	if len(failed) > 0 {
		fmt.Printf("FAILURES (%d targets failed):\n", len(failed))

		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}

		os.Exit(1)
	}

	fmt.Printf("SUCCESS: All %d fuzz targets passed with 0 panics and 0 errors!\n", len(targets))
}
