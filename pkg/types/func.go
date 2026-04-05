package types

import (
	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"k8s.io/client-go/tools/cache"
)

type NewReconcilerFunc func(
	kube *kubeclient.Kubeclient,
	inf cache.SharedIndexInformer,
	ev *event.Event,
) domain.Reconciler
