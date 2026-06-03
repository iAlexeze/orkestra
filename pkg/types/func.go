package types

import (
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"k8s.io/client-go/tools/cache"
)

type NewReconcilerFunc func(
	kube kubeclient.KubeClient,
	inf cache.SharedIndexInformer,
	ev event.Recorder,
) domain.Reconciler
