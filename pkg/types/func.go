package types

import (
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"k8s.io/client-go/tools/cache"
)

type NewReconcilerFunc func(
	kube *kubeclient.Kubeclient,
	inf cache.SharedIndexInformer,
	ev *event.Event,
) domain.Reconciler
