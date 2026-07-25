// Package external executes declarative external calls (HTTP today, protocol
// clients in future) and injects results into the resolver context under
// .external.<name>. Used by both the reconciler (at reconcile time) and the
// gateway webhook (at admission time).
package external

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Run executes all declared external calls sequentially and returns a new
// resolver with results injected under .external.<name>.
//
// Calls run in declaration order. Each call can reference earlier calls'
// results in its own URL or body template expressions.
//
// gvk is the CRD identifier used for metrics labelling only.
// Returns the enriched resolver. The original is unchanged.
// Returns an error only when continueOnError is false and a call fails.
func Run(
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
		if !orktypes.EvaluateWhen(resolver.Data(), call.Conditions, call.AnyOf, resolver.TemplateEvaluator()) {
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

		resolvedURL, err := resolver.Resolve(call.URL)
		if err != nil {
			return resolver, fmt.Errorf("external[%d].url: %w", i, err)
		}
		resolvedBody, err := resolver.Resolve(call.Body)
		if err != nil {
			return resolver, fmt.Errorf("external[%d].body: %w", i, err)
		}
		resolvedToken, err := resolver.Resolve(ExpandEnv(call.Token))
		if err != nil {
			return resolver, fmt.Errorf("external[%d].token: %w", i, err)
		}

		result := executeHTTPCall(ctx, call, resolvedURL, resolvedBody, resolvedToken)

		entry := map[string]interface{}{}
		if result.Body != "" {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(result.Body), &parsed); err == nil {
				for k, v := range parsed {
					entry[k] = v
				}
			}
		}
		// HTTP meta fields are set after JSON parsing so they are never
		// overwritten by a body key with the same name (e.g. {"status":"ok"}).
		entry["status"] = result.Status
		entry["body"] = result.Body
		entry["error"] = result.Error
		entry["called"] = "true"
		results[call.Name] = entry

		metrics.RecordExternalCall(gvk, call.Name, resolvedURL, result.DurationSeconds, result.Error, result.StatusCode)

		// Inject before any early return so status fields can reference this
		// call's result even when continueOnError is false.
		resolver = resolver.WithExternal(results)

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
	}

	return resolver, nil
}
