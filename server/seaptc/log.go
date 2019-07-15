package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	traceIDPrefix = os.ExpandEnv("projects/$GOOGLE_CLOUD_PROJECT/traces/")
	jsonLogging   = os.Getenv("GAE_SERVICE") != ""
)

func newContextWithTraceID(ctx context.Context, r *http.Request) context.Context {
	// Use App Engine traceid for correlation in the log viewer.
	s := r.Header.Get("X-Cloud-Trace-Context")
	if i := strings.IndexByte(s, '/'); i > 0 {
		return context.WithValue(ctx, "traceID", traceIDPrefix+s[:i])
	}
	return ctx
}

// logf logs the given message.
//
// If running on App Engine, the message is logged in a JSON format understood
// by StackDriver.
//
// See https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
// for list of severity levels.
func logf(ctx context.Context, severity string, format string, args ...interface{}) {
	if jsonLogging {
		m := map[string]string{
			"severity": severity,
			"time":     time.Now().Format(time.RFC3339Nano),
			"message":  fmt.Sprintf(format, args...),
		}

		if traceID, ok := ctx.Value("traceid").(string); ok {
			m["logging.googleapis.com/trace"] = traceID
		}
		p, _ := json.Marshal(m)
		p = append(p, '\n')
		os.Stderr.Write(p)
	} else {
		log.Printf(format, args...)
	}
}
