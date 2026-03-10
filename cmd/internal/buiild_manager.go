package internal

import (
	"context"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/health"
	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/kontroller"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/manager"
	"github.com/ialexeze/orkestra/pkg/queue"
	"github.com/ialexeze/orkestra/pkg/registry"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

type startupCfg struct {
	kontroller *kontroller.DependencyKontroller
	event      *event.Event
	kube       *kubeclient.Kubeclient
	manager    *manager.Manager
	komp       *[]domain.Komponent
}

func buildManager(kfg *konfig.Konfig, ctx context.Context) *startupCfg {
	// crd registry
	crdRegistry := registry.NewCRDRegistry(kfg.CRDRegistry().Mode, kfg.CRDRegistry().Path)

	// scheme registry
	scheme, err := registry.NewSchemeRegistry(crdRegistry)
	if err != nil {
		logger.Fatal().Err(err).Msg("scheme creation error")
	}

	// health
	hs := health.NewHealthServer(kfg)

	// kube
	kube := kubeclient.NewKubeclient(kubeclient.Config{
		Kubeconfig: kfg.Cluster().KubekonfigPath,
		Masterurl:  kfg.Cluster().MasterURL,
		Scheme:     scheme,
	})

	// events
	ev := event.NewEvent(kube)

	// queue
	wq := queue.NewWorkqueue()

	// provider
	provider := kube.ClientProvider()

	// Register CRD clients to provider - for automatic client  and informer generation
	for _, crd := range crdRegistry.CRDs {
		var object runtime.Object
		var list runtime.Object

		if kfg.YamlMode() {
			object = crd.ObjectYamlMode()
			list = crd.ListObjectYamlMode()
		} else if kfg.GoMode() {
			object = crd.ObjectGoMode
			list = crd.ListObjectGoMode
		}

		// 3. Register in kontroller registry
		logger.Debug().Str("GVK", utils.SetGroupVersionKindObj(crd.GroupVersionKind)).Msg("registering CRD")

		provider.Register(object, func(k *kubeclient.Kubeclient) (informer.GenericClient, error) {
			return k.NewClient(list, kubeclient.CRDInfo{
				Kind:         crd.Kind,
				Group:        crd.Group,
				Version:      crd.Version,
				APIPath:      crd.APIPath,
				GroupVersion: crd.GroupVersion,
				Plural:       crd.Plural,
				Namespace:    crd.Namespace,
				Namespaced:   crd.Namespaced,
			})
		})
	}

	// Create shared informer factory
	infFactory := informer.SharedInformerFactory(
		provider,
		wq,
		scheme,
		kfg.Cluster().Namespace,
		kfg.Cluster().DefaultResync,
	)

	// Kontroller Registry
	reg := kontroller.NewKontrollerRegistry()

	// Register CRDs to kontroller registry
	logger.Info().Msg("registering CRDs...")
	for _, crd := range crdRegistry.CRDs {
		var object runtime.Object
		if kfg.YamlMode() {
			object = crd.ObjectYamlMode()
		} else if kfg.GoMode() {
			object = crd.ObjectGoMode
		}

		// 1. Create informer
		inf := infFactory.For(object, ctx, informer.Options{
			Name:   crd.Kind,
			Resync: crd.Resync,
		})

		// 2. Create reconciler
		rec := crd.Reconciler(kube, inf, ev)

		// 3. Register in kontroller registry
		logger.Debug().Str("GVK", utils.SetGroupVersionKindObj(crd.GroupVersionKind)).Msg("registering CRD")
		reg.Register(
			utils.SetGroupVersionKindObj(crd.GroupVersionKind),
			crd,
			inf,
			rec,
		)
	}

	// Add all komponents
	komponents := []domain.Komponent{
		hs,
		kube,
		ev,
		wq,
		infFactory,
	}

	// kontroller manager
	ktrl := kontroller.NewDependencyKontroller(
		kube,
		infFactory,
		reg,
		ev,
		wq,
		hs,
		kfg.Cluster().DefaultWorkers,
		kfg.CRDRegistry().MaxQueueDepth,
		registry.NewDependencyGraph(crdRegistry),
		&kontroller.BannerKonfig{
			AllCRDs:    crdRegistry.List(),
			Konfig:     kfg,
			Komponents: komponents,
			Leader:     "",
		},
	)

	// Add kontroller
	komponents = append(komponents, ktrl)

	// manager
	mgr := manager.NewManager(kfg.Cluster().DefaultResync)
	mgr.Register(komponents) // Register all manager komponents

	return &startupCfg{
		event:      ev,
		kontroller: ktrl,
		kube:       kube,
		komp:       &komponents,
		manager:    mgr,
	}
}
