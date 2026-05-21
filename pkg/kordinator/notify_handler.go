package kordinator

import (
	"encoding/json"
	"net/http"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/notification"
)

// BuildNotifyHandler returns an http.HandlerFunc that receives a notification
// Event from the runtime and dispatches it to the configured team channels.
//
// The gateway is the single point for SMTP/Slack dispatch — the runtime only
// POSTs pre-built events after its own throttle check.
func BuildNotifyHandler(kat *katalog.Katalog) http.HandlerFunc {
	notifier := notification.NewDirectNotifier(kat)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var ev notification.Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		_ = notifier.Dispatch(r.Context(), ev)
		w.WriteHeader(http.StatusAccepted)
	}
}
