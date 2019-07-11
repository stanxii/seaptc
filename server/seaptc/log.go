package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var traceIDPrefix = os.ExpandEnv("projects/$GOOGLE_CLOUD_PROJECT/traces/")

// logf formats args according to the format specifier and writes the result to
// stderr in the JSON format supported by StackDriver.
//
// See https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
// for list of severity levels.
func logf(severity string, header http.Header, format string, args ...interface{}) {
	m := map[string]string{
		"severity": severity,
		"time":     time.Now().Format(time.RFC3339Nano),
		"message":  fmt.Sprintf(format, args...),
	}

    // Use App Engine traceid for correlation in the log viewer.
	s := header.Get("X-Cloud-Trace-Context")
	if i := strings.IndexByte(s, '/'); i > 0 {
		m["logging.googleapis.com/trace"] = traceIDPrefix + s[:i]
	}

	p, _ := json.Marshal(m)
	p = append(p, '\n')
	os.Stderr.Write(p)
}
