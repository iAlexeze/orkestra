// pkg/reconciler/run_external.go
//
// External HTTP call dispatch — runs before resource groups in runTemplateReconcile.
//
// Calls are executed sequentially in declaration order. Results are injected
// into the resolver context via resolver.WithExternal() so subsequent
// template expressions and when: conditions can reference them.
//
// Call sequence in runTemplateReconcile:
//
//  1. resolver = NewResolver(ctx, obj)
//  2. resolver = resolver.WithCross(ReadCross(...))    ← cross-CRD first
//  3. resolver, err = runExternal(ctx, resolver, ...)  ← HTTP calls second
//  4. runDeployments, runServices, ... etc.            ← resources use results
//
// Results in template context:
//
//	.external.<name>.status   → HTTP status code ("200", "404", "503")
//	.external.<name>.body     → response body (first 4096 bytes)
//	.external.<name>.error    → error message on failure
//	.external.<name>.called   → "true" if the call was made
//
// When: conditions on deployments can gate on these:
//
//	when:
//	  - field: external.health-check.status
//	    equals: "200"
package reconciler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

const (
	// maxBodyBytes is the maximum bytes read from an HTTP response body.
	// Large responses are truncated — template expressions rarely need full bodies.
	maxBodyBytes = 4096

	// defaultExternalTimeout is the per-call timeout when none is declared.
	defaultExternalTimeout = 10 * time.Second
)

// runExternal executes all declared external HTTP calls and returns a new
// resolver with the results injected under .external.*.
//
// Calls run sequentially — later calls can reference earlier results via
// template expressions in their URL or body fields.
//
// Returns the enriched resolver. The original resolver is unchanged.
// Returns an error only when continueOnError is false and a call fails.
func runExternal(
	ctx context.Context,
	gvk string,
	resolver *orktmpl.Resolver,
	calls []orktypes.ExternalCallSpec,
) (*orktmpl.Resolver, error) {
	if len(calls) == 0 {
		return resolver, nil
	}

	log := logger.FromContext(ctx)
	results := make(map[string]interface{}, len(calls))

	for i, call := range calls {
		// Evaluate when: conditions before making the call.
		// A skipped call produces a "called": "false" result — not an error.
		if !orktypes.EvaluateWhen(resolver.Data(), call.Conditions, call.AnyOf) {
			results[call.Name] = map[string]interface{}{
				"status": "",
				"body":   "",
				"error":  "",
				"called": "false",
			}
			log.Debug().
				Str("call", call.Name).
				Int("index", i).
				Msg("external call skipped — when: conditions not met")
			continue
		}

		// Resolve template expressions in the call spec
		resolvedURL, err := resolver.Resolve(call.URL)
		if err != nil {
			return resolver, fmt.Errorf("external[%d].url: %w", i, err)
		}
		resolvedBody, err := resolver.Resolve(call.Body)
		if err != nil {
			return resolver, fmt.Errorf("external[%d].body: %w", i, err)
		}
		resolvedToken, err := resolver.Resolve(expandEnv(call.Token))
		if err != nil {
			return resolver, fmt.Errorf("external[%d].token: %w", i, err)
		}

		// Execute the call
		result := executeHTTPCall(ctx, call, resolvedURL, resolvedBody, resolvedToken)

		results[call.Name] = map[string]interface{}{
			"status": result.Status,
			"body":   result.Body,
			"error":  result.Error,
			"called": "true",
		}

		metrics.RecordExternalCall(gvk, call.Name, resolvedURL, result.DurationSeconds, result.Error, result.StatusCode)

		if result.Error != "" {
			log.Warn().
				Str("call", call.Name).
				Str("url", resolvedURL).
				Str("error", result.Error).
				Msg("external call failed")

			if !call.ContinueOnError {
				return resolver, fmt.Errorf("external call %q failed: %s", call.Name, result.Error)
			}
		} else {
			log.Debug().
				Str("call", call.Name).
				Str("url", resolvedURL).
				Str("status", result.Status).
				Msg("external call succeeded")
		}

		// Inject accumulated results so far — later calls can reference earlier ones.
		// Each iteration creates a new resolver with the growing results map.
		resolver = resolver.WithExternal(results)
	}

	return resolver, nil
}

// executeHTTPCall makes one HTTP request and returns the result.
// Never panics. Returns an error result on failure.
func executeHTTPCall(
	ctx context.Context,
	spec orktypes.ExternalCallSpec,
	url, body, token string,
) orktypes.ExternalCallResult {
	timeout := defaultExternalTimeout
	if spec.Timeout != "" {
		if d, err := time.ParseDuration(spec.Timeout); err == nil {
			timeout = d
		}
	}

	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(callCtx, method, url, bodyReader)
	if err != nil {
		return orktypes.ExternalCallResult{
			Error:  fmt.Sprintf("building request: %v", err),
			Called: "true",
		}
	}

	// Set headers
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return orktypes.ExternalCallResult{
			Error:  fmt.Sprintf("executing request: %v", err),
			Called: "true",
		}
	}
	defer resp.Body.Close()

	// Read body — capped at maxBodyBytes
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return orktypes.ExternalCallResult{
			Status: fmt.Sprintf("%d", resp.StatusCode),
			Error:  fmt.Sprintf("reading response body: %v", err),
			Called: "true",
		}
	}

	durationSeconds := time.Since(start).Seconds()

	statusCode := fmt.Sprintf("%d", resp.StatusCode)
	callErr := ""

	// Validate expected status if declared
	if spec.ExpectedStatus > 0 && resp.StatusCode != spec.ExpectedStatus {
		callErr = fmt.Sprintf("expected status %d, got %d", spec.ExpectedStatus, resp.StatusCode)
	} else if resp.StatusCode >= 400 && spec.ExpectedStatus == 0 {
		// Default: 4xx/5xx are errors
		callErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return orktypes.ExternalCallResult{
		Status:          statusCode,
		StatusCode:      resp.StatusCode,
		Body:            string(rawBody),
		Error:           callErr,
		Called:          "true",
		DurationSeconds: durationSeconds,
	}
}

// expandEnv expands $VAR and ${VAR} environment variable references.
// Used for token fields so secrets stay out of Katalog YAML.
func expandEnv(s string) string {
	return os.ExpandEnv(s)
}
