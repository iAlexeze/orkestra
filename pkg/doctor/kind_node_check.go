package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// nodeList represents the output of `kubectl get nodes -o json`
type nodeList struct {
	Items []struct {
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

// waitForNodesReady polls kubectl until all nodes report Ready or timeout expires.
func waitForNodesReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("nodes not ready after %s", timeout)
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "-o", "json")
			output, err := cmd.Output()
			if err != nil {
				// cluster may still be starting; continue waiting
				continue
			}

			var nodes nodeList
			if err := json.Unmarshal(output, &nodes); err != nil {
				continue // invalid JSON? retry
			}

			if len(nodes.Items) == 0 {
				continue
			}

			allReady := true
			for _, node := range nodes.Items {
				ready := false
				for _, cond := range node.Status.Conditions {
					if cond.Type == "Ready" && cond.Status == "True" {
						ready = true
						break
					}
				}
				if !ready {
					allReady = false
					break
				}
			}

			if allReady {
				return nil
			}
		}
	}
}
