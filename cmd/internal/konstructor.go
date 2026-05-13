// cmd/internal/konstruct.go
//
// konstructOrkestra — the complete Orkestra runtime registry.
//
// This file is the single place where all runtime komponents are assembled.
// It is the equivalent of a dependency injection container — every komponent
// is created here, every dependency is threaded here, and nothing is started
// here. Starting happens in orkestra.Start() in declaration order.
//
// ── Architecture overview ─────────────────────────────────────────────────
//
//   Katalog (YAML)
//       │
//       ▼
//   merger → katalog.Katalog          One Katalog per operator binary.
//       │                             Holds all CRD declarations, reconciler
//       │                             configs, validation/mutation rules.
//       │
//       ▼
//   kubeclient.Kubeclient             REST config, dynamic client, typed
//       │                             clientset. Started first — everything else
//       │                             needs it.
//       │
//       ├──► ClientProvider           One REST client constructor per CRD.
//       │                             Deferred — constructed on first use.
//       │
//       ├──► SharedInformerFactory    One SharedIndexInformer per CRD.
//       │        │                    Starts watching the API server on Start().
//       │        │                    Routes watch events into per-CRD workqueues.
//       │        │
//       │        └──► per-CRD informer (cache.SharedIndexInformer)
//       │                 Holds all CR instances in memory.
//       │                 Zero API calls for reads after initial sync.
//       │
//       ├──► ProviderRegistry         AWS, MongoDB, Stripe — external infra providers.
//       │                             Registered before factory closures so all
//       │                             reconcilers share the same registry.
//       │
//       ├──► ResourceKatalog          Maps GVK → (CRD, informer, reconcilerFactory).
//       │    (ktrlRegistry)           Also implements KatalogRegistry for cross-CRD
//       │                             observation via GetInformerByName.
//       │
//       ├──► per-CRD reconciler factory closure
//       │        Captures: crdInfo, infCopy, ev, kube, anyHooks, newObj,
//       │                  providerRegistry, ktrlRegistry
//       │        Called by startCRDWorkers after orkestra.Start().
//       │        Returns a *GenericReconciler[T] ready to process items.
//       │
//       ├──► DependencyKordinator     Starts CRD workers in topological order.
//       │                             Waits for dependencies to meet their declared
//       │                             condition (started | healthy) before starting
//       │                             dependent workers.
//       │
//       └──► HealthServer             HTTP server for health, Katalog API, and
//                                     Control Center. Routes registered before Start().
//
// ── Reconcile loop (per CR item dequeued) ────────────────────────────────
//
//   workqueue.Get(key)
//       │
//       ▼
//   GenericReconciler.Reconcile(ctx, key)
//       │
//       ├── informer.GetIndexer().GetByKey(key)   (in-memory, zero API call)
//       │
//       ├── ensureFinalizers / ensureManagedLabel / ensureManagedAnnotations
//       │
//       ├── handleDeletion  (if DeletionTimestamp set)
//       │     └── runTemplateOnDelete → provider.Delete → removeFinalizers
//       │
//       └── reconcileImpl
//             ├── mutation  (apply defaults)
//             ├── validation (deny violations halt, warn violations log)
//             │
//             ├── OnReconcile hook (Go typed hook, if registered)
//             │
//             └── runTemplateReconcile  (declarative path)
//                   ├── 1. NewResolver(obj)           .spec.*, .status.*, .metadata.*
//                   ├── 2. readCross(decls)           .cross.<kind>.status.*
//                   │         └── katalogRegistry.GetInformerByName(kind)
//                   │               └── informer.GetIndexer().GetByKey(key)
//                   │                     zero API calls for same-binary CRDs
//                   ├── 3. runExternal(calls)         .external.<n>.status, .body
//                   │         └── http.Do(req) per call, sequential
//                   ├── 4. forEach expansion           N sources → N reconciles
//                   ├── 5. runResourceGroup(onCreate)
//                   │         runDeployments, runServices, runSecrets (once:),
//                   │         runConfigMaps, runServiceAccounts, runJobs, runCronJobs
//                   ├── 6. runResourceGroup(onReconcile) — same, update=true
//                   └── 7. runProviders(blocks)        aws:, mongodb:, stripe:
//                               └── provider.Reconcile(ctx, req) per block
//
//   After reconcileImpl:
//       patchStatusWithChildren(ctx, obj, err)
//           ├── ReadChildren → .children.*  (API server, parallel, RV="0")
//           ├── resolveStatusFields(when:, anyOf:, template expressions)
//           └── PATCH /status

package internal

import (
	"context"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/certmanager"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/informer"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kordinator"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	ork "github.com/orkspace/orkestra/pkg/orkestra"
	"github.com/orkspace/orkestra/pkg/queue"
	"github.com/orkspace/orkestra/pkg/reconciler"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/orkspace/orkestra/pkg/webhook"
	"k8s.io/client-go/tools/cache"
)

// orkestraKfg is the assembled runtime — returned to main.go so it can call
// orkestra.Start(ctx) and block until shutdown.
type orkestraKfg struct {
	konfig   *konfig.Konfig
	katalog  *katalog.Katalog
	komp     *[]domain.Komponent
	event    *event.Event
	kube     *kubeclient.Kubeclient
	kord     *kordinator.DependencyKordinator
	orkestra *ork.Orkestra

	// TLS cert paths — set only when certs were auto-generated by ensureSecurity.
	// Empty strings when the user provided explicit TLS_CERT/TLS_KEY env vars.
	// Used by the shutdown hook in Konduct to clean up temp files.
	tlsCert string
	tlsKey  string
}

// konstructOrkestra wires the entire Orkestra runtime.
//
// Nothing is started here. Every component is constructed and threaded together
// as closures and pointers. orkestra.Start() calls komponent.Start() in
// registration order, and komponent.Stop() in reverse order on shutdown.
//
// The method is intentionally long — this is the one place where all wiring
// is visible. Splitting it would scatter the dependency graph across files
// and make it harder to reason about startup order.
func konstructOrkestra(kfg *konfig.Konfig, m *merger.Merger, ctx context.Context) *orkestraKfg {

	// ── 1. Katalog ────────────────────────────────────────────────────────────
	// Loads and validates the YAML Katalog. After this point, kat.Enabled()
	// returns only CRDs that passed schema validation and are not disabled.
	// Invalid CRDs are logged and excluded — they do not block the operator.
	kat := katalog.NewKatalog(kfg, m)

	if registryURL := kfg.RegistryConfig().RegistryURL; registryURL != "" {
		m.SetRegistryURL(registryURL)
		logger.Info().Str("registry", registryURL).Msg("registry URL configured from ORK_REGISTRY")
	}

	// ── 2. Scheme ─────────────────────────────────────────────────────────────
	// Each CRD type (e.g. *PipelineList) must be registered with the scheme so
	// the REST client knows how to decode API server responses. For dynamic CRDs
	// (unstructured mode), this is a no-op — they use the dynamic client.
	scheme, err := katalog.NewSchemeRegistry(kat)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to build scheme registry")
	}

	// ── 3. Core komponents ────────────────────────────────────────────────────
	// Created here, started later by orkestra in registration order.

	kube := kubeclient.NewKubeclient(kfg, scheme)

	// kubeclient is started immediately — the informer factory's missing-CRD
	// check needs the REST config during construction, before orkestra.Start().
	if err := kube.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start kubeclient")
	}

	var tlsCert, tlsKey string
	var certMgr certmanager.Manager
	if utils.IsRunningInCluster() {
		tlsCert, tlsKey, certMgr, err = ensureSecurity(ctx, kfg, kat, kube)
		if err != nil {
			logger.Fatal().Err(err).Msg("security setup failed")
		}

		// Make generated certs available to the WebhookServer via kfg.
		if tlsCert != "" {
			logger.Debug().
				Str("cert_file", tlsCert).
				Str("cert_key", tlsKey).
				Msg("passing generated TLS cert to webhook server")
			kfg.Security().Webhooks.TLSCert = tlsCert
			kfg.Security().Webhooks.TLSKey = tlsKey
		}
	}

	// HealthServer — HTTP-only (health, readiness, metrics, Katalog API routes).
	// Routes registered below before Start() binds the port.
	hs := health.NewHealthServer(kfg)

	// WebhookServer — HTTPS admission and conversion webhook surface.
	// Starts after HealthServer so /ready is live before webhook registration.
	ws := webhook.NewWebhookServer(kube.Clientset(), kat, kfg)
	if certMgr != nil {
		ws.SetCertManager(certMgr)
	}

	// Event recorder — surfaces notable state changes to the Kubernetes event stream
	// (visible via kubectl describe). Shared by all controllers that emit events.
	ev := event.NewEvent(kube)

	// Default work queue — rate-limiting reconcile queue shared by controllers that
	// do not need a dedicated queue. Bounded to prevent runaway reconcile storms.
	defaultWq := queue.NewWorkqueue()

	// Queue registry — maps GVK strings to dedicated work queues. Controllers
	// register here so that cross-controller enqueues are dispatched correctly.
	queueRegistry := queue.NewQueueRegistry()

	// ── 4a. REST client provider ──────────────────────────────────────────────
	// Associates each CRD type with a constructor that builds a typed REST
	// client. The constructor is deferred — called on first informer use.
	// Dynamic CRDs skip this — they use the dynamic client directly.
	provider := kube.NewClientProvider()

	for _, crd := range kat.Enabled() {
		crd := crd
		if crd.IsDynamic() {
			continue
		}
		object, list := crd.GetRuntimeObjects()
		logger.Debug().Str("gvk", crd.GVK().String()).Msg("registering CRD client provider")

		provider.Register(object, func(k *kubeclient.Kubeclient) (informer.GenericClient, error) {
			return k.NewClient(list, kubeclient.CRDInfo{
				Kind:         crd.APITypes.Kind,
				Group:        crd.APITypes.Group,
				Version:      crd.APITypes.Version,
				APIPath:      crd.APITypes.APIPath,
				GroupVersion: crd.GroupVersion,
				Plural:       crd.APITypes.Plural,
				Namespace:    crd.Namespace,
				Namespaced:   crd.IsNamespaced(),
			})
		})
	}

	// ── 4b. Shared informer factory ───────────────────────────────────────────
	// Creates one SharedIndexInformer per CRD. On Start(), each informer opens
	// a watch against the API server and populates its in-memory cache.
	// Watch events are routed into per-CRD workqueues via handleEvent.
	infFactory := informer.SharedInformerFactory(
		provider,
		kube.RestConfig(),
		queueRegistry,
		defaultWq,
		scheme,
		kfg,
	)

	// ── 4c. Provider registry ─────────────────────────────────────────────────
	// External infrastructure providers (AWS, MongoDB, etc.).
	// Must be built BEFORE the factory loop so all reconciler closures capture
	// the same fully-initialised registry. loadProviders is non-fatal —
	// unavailable providers log a warning and the operator starts regardless.
	providerRegistry := loadProviders(ctx, kat)

	// One ProviderStats per CRD — shared between GenericReconciler (writes on each
	// provider call) and BuildCRDInfoHandler (reads for the /katalog/{crd} response).
	// Only created for CRDs that declare provider blocks — others get nil.
	providerStatsMap := make(map[string]*health.ProviderStats)
	for _, crd := range kat.Enabled() {
		if crd.HasProviders() {
			providerStatsMap[crd.GVK().String()] = health.NewProviderStats()
		}
	}

	// ── 4d. Kordinator registry + per-CRD wiring ──────────────────────────────
	// ktrlRegistry maps GVK → (CRDEntry, SharedIndexInformer, ReconcilerFactory).
	// It also implements reconciler.KatalogRegistry via GetInformerByName,
	// enabling cross-CRD observation with zero API server calls.
	ktrlRegistry := kordinator.NewKordinatorRegistry()

	// ── 4e. CRD health map ────────────────────────────────────────────────────
	// One CRDHealth per CRD — shared between the DependencyKordinator
	// (which updates it on each reconcile) and the HTTP health routes
	// (which read it on each request). All three reference the same pointers.
	crdHealthMap := make(map[string]*kordinator.CRDHealth)
	for _, crd := range kat.Enabled() {
		gvk := crd.GVK().String()
		crdHealthMap[gvk] = kordinator.NewCRDHealth(crd.Name)
	}

	logger.Debug().Msg("wiring CRDs into kordinator registry...")

	finalizers := kfg.Finalizers()
	for _, crd := range kat.Enabled() {
		crd := crd
		gvk := crd.GVK().String()
		crd.Workers = crd.SetWorkers(kfg.Katalog().DefaultWorkers)

		object, _ := crd.GetRuntimeObjects()

		wq := queueRegistry.Register(gvk, crd.SetMaxQueueDepth(kfg.Katalog().DefaultMaxQueueDepth))

		// compute selectors
		labelSelector := orktypes.SelectorMap(crd.LabelSelector).String()
		fieldSelector := orktypes.SelectorMap(crd.FieldSelector).String()

		opts := informer.Options{
			Name:          crd.APITypes.Kind,
			Resync:        crd.Resync,
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		}
		if crd.DefaultQueue() {
			opts.Wq = nil // use the shared default queue
		} else {
			opts.Wq = wq
		}

		// ── Namespace filter — Tier 1 (scope ListerWatcher) + Tier 2 (pre-enqueue) ──
		// Tier 2 is always registered when namespace rules exist.
		// Tier 1 scopes the ListerWatcher to a single namespace when allowedNamespaces
		// has exactly one entry — the informer never sees events from other namespaces.
		if len(crd.AllowedNamespaces) > 0 || len(crd.RestrictedNamespaces) > 0 {
			filter := &informer.NamespaceFilter{
				AllowedNamespaces:    []string(crd.AllowedNamespaces),
				RestrictedNamespaces: []string(crd.RestrictedNamespaces),
			}
			if filter.IsSingleNamespace() {
				opts.Namespace = filter.SingleNamespace()
				logger.Debug().
					Str("crd", crd.APITypes.Kind).
					Str("namespace", opts.Namespace).
					Msg("informer: namespace-scoped watch (Tier 1)")
			}
			infFactory.RegisterNamespaceFilter(gvk, filter)
			logger.Debug().
				Str("crd", crd.APITypes.Kind).
				Str("filter", informer.NamespaceFilterSummary(filter)).
				Msg("informer: namespace filter registered (Tier 2)")
		}

		// Choose typed or dynamic informer.
		// Dynamic CRDs use *unstructured.Unstructured — no Go type needed.
		// Typed CRDs use the registered concrete Go type for type-safe access.
		var inf cache.SharedIndexInformer

		// For dynamic CRDs, use opts.Namespace from the Tier 1 filter when set;
		// otherwise fall back to crd.Namespace (operator-level namespace setting).
		dynNamespace := crd.Namespace
		if opts.Namespace != "" {
			dynNamespace = opts.Namespace
		}

		logger.Debug().
			Bool("dynamic:", crd.IsDynamic()).
			Msgf("[DEBUG] CRD %s: location = %q\n", crd.APITypes.Kind, crd.APITypes.Location)
		if crd.IsDynamic() {
			lw := kube.NewDynamicListerWatcher(kubeclient.CRDInfo{
				Kind:         crd.APITypes.Kind,
				Group:        crd.APITypes.Group,
				Version:      crd.APITypes.Version,
				APIPath:      crd.APITypes.APIPath,
				GroupVersion: crd.GroupVersion,
				Plural:       crd.APITypes.Plural,
				Namespace:    dynNamespace,
				Namespaced:   crd.IsNamespaced(),
			}, kubeclient.ListOptions{
				LabelSelector: labelSelector,
				FieldSelector: fieldSelector,
			})
			inf = infFactory.ForListerWatcher(lw, object, ctx, opts)
		} else {
			inf = infFactory.For(object, ctx, opts)
		}

		finalizers = append(finalizers, crd.OperatorBox.Finalizers...)

		infCopy := inf

		// Build the reconciler factory.
		// For default: true CRDs — GenericReconciler interprets the Katalog declaratively.
		// For default: false CRDs — a custom Constructor is required.
		//
		// The factory is a closure — it captures all values at construction time
		// and is called by startCRDWorkers after informers are synced.
		// Each call returns a fresh reconciler instance for one worker goroutine.
		var factory func() domain.Reconciler

		if crd.DefaultReconcile() {
			objCopy := object

			var anyHooks domain.AnyReconcileHooks
			if crd.OperatorBox.HookFactory != nil {
				anyHooks = crd.OperatorBox.HookFactory()
			}

			logger.Debug().Str("gvk", gvk).Msg("wiring GenericReconciler factory")

			pStats := providerStatsMap[gvk]
			factory = func() domain.Reconciler {
				return reconciler.NewGenericReconciler(
					crd,
					infCopy,
					ev,
					kube,
					anyHooks,
					func() domain.Object {
						return objCopy.DeepCopyObject().(domain.Object)
					},
					ktrlRegistry,     // cross-CRD informer lookup via GetInformerByName
					crdHealthMap,     // cross-CRD health map via HealthProvider
					providerRegistry, // aws:, mongodb:, etc. block dispatch
					pStats,           // per-CRD provider error rate tracking
				)
			}
		} else {
			if crd.OperatorBox.Constructor == nil {
				logger.Fatal().
					Str("gvk", gvk).
					Msg("reconciler.default is false but no Constructor provided")
			}

			logger.Debug().Str("gvk", gvk).Msg("wiring custom reconciler factory")

			factory = func() domain.Reconciler {
				return crd.OperatorBox.Constructor(kube, infCopy, ev)
			}
		}

		// Register informs the DependencyKordinator which informer and factory
		// belong to this CRD. Workers are not started yet — that happens in Start().
		ktrlRegistry.Register(gvk, crd, inf, factory)
		logger.Debug().Str("gvk", gvk).Msg("CRD registered")
	}

	// ── 5. HTTP routes ───────────────────────────────────────────────────────
	// All routes registered before hs.Start() — the mux is shared.
	//
	// Per-CRD routes:
	//   /katalog/{crd}/health       		→ 200 healthy, 503 degraded
	//   /katalog/{crd}              		→ CRD config + live reconcile stats
	//	 /katalog/{crd}/raw			 		→ the user's config
	//   /katalog/{crd}/enriched	 		→ the runtime config
	//   /katalog/{crd}/cr           		→ all CR instances (informer cache, <1ms)
	//   /katalog/{crd}/cr/{ns}/{n}  		→ CR detail + children (watch cache, <50ms)
	//   /katalog/{crd}/cr/{...}/events 	→ recent events (watch cache, <50ms)
	//
	// Aggregate:
	//	 /katalog/raw				 		→ the user's katalog config
	//	 /katalog/enriched				 	→ the runtime katalog config
	//   /katalog                    		→ all CRDs, dependency graph, health summary
	deletionProtectedCRDs := kat.DeletionProtectedCRDNames()
	for _, crd := range kat.Enabled() {
		gvk := crd.GVK().String()
		crdHealth := crdHealthMap[gvk]
		crdName := strings.ToLower(crd.Name)

		entry, _ := ktrlRegistry.Get(gvk)
		inf := entry.Informer

		if !crd.IsEnabledAllEndpoints() {
			continue
		}

		crdKey := crd.APITypes.Plural + "." + crd.APITypes.Group
		_, isDeletionProtected := deletionProtectedCRDs[crdKey]

		if crd.IsHealthEnabled() {
			hs.Register(
				"/katalog/"+crdName+"/health",
				kordinator.BuildCRDHealthHandler(crd, kfg, inf, crdHealth),
			)
		}

		if crd.IsInfoEnabled() {
			hs.Register(
				"/katalog/"+crdName,
				kordinator.BuildCRDInfoHandler(
					crd, kfg, inf, crdHealth,
					ws.GetConversionStats(),
					ws.GetAdmissionStats(),
					ws.GetProtectionStats(),
					ws.GetWebhookStats(),
					providerStatsMap[gvk],
					ws.GetNamespaceStats(),
					isDeletionProtected,
					kat.IsNamespaceProtectionEnabled(),
					kat.IsConversionEnabled(),
					kat.IsAdmissionEnabled(),
				),
			)
			hs.Register(
				"/katalog/"+crdName+"/cr",
				kordinator.BuildCRListHandler(crd, inf),
			)
			hs.Register(
				"/katalog/"+crdName+"/cr/",
				kordinator.BuildCRDetailAndEventsHandler(crd, inf, kube),
			)
		}

		// Register raw and enriched CRD definition endpoint
		hs.Register(
			"/katalog/"+crdName+"/raw",
			kordinator.BuildCRDRawHandler(m, crd.Name),
		)
		hs.Register(
			"/katalog/"+crdName+"/enriched",
			kordinator.BuildCRDEnrichedHandler(kat, crd.Name),
		)

		logger.Debug().
			Str("health", "/katalog/"+crdName+"/health").
			Str("info", "/katalog/"+crdName).
			Str("raw", "/katalog/"+crdName+"/raw").
			Str("enriched", "/katalog/"+crdName+"/enriched").
			Msg("registered CRD routes")
	}

	orkHealth := kordinator.NewOrkestraHealth()
	hs.Register("/katalog/raw", kordinator.BuildRawKatalogHandler(m))
	hs.Register("/katalog/enriched", kordinator.BuildEnrichedKatalogHandler(kat))
	hs.Register("/katalog", kordinator.BuildKatalogHandler(kat, kfg, ktrlRegistry, crdHealthMap, orkHealth))

	// ── 6. Dependency kordinator ──────────────────────────────────────────────
	// Starts CRD workers in topological order defined by the dependency graph.
	// For each CRD, waits until all declared dependsOn CRDs meet their
	// condition (started | healthy) before calling factory() and starting workers.
	//
	// Worker lifecycle:
	//   Start() → wait for informer sync → call factory() per worker → run loop
	//   Shutdown → drain queue → stop workers → remove from active set
	kord := kordinator.NewDependencyKordinator(
		kube,
		infFactory,
		ktrlRegistry,
		ev,
		hs,
		queueRegistry,
		defaultWq,
		crdHealthMap,
		orkHealth,
		kfg.Katalog().DefaultWorkers,
		katalog.NewDependencyGraph(kat),
		kfg.Katalog().ShutdownTimeout,
	)

	// ── 7. Komponent list ─────────────────────────────────────────────────────
	// Start order: each komponent must start after its dependencies.
	// Stop order: reverse of start order (automatic).
	//
	// HealthServer starts first so it can serve /ready during startup.
	// Kubeclient is already started above but is still registered so
	// orkestra manages its Stop().
	komponents := []domain.Komponent{
		hs,            // 1. HTTP server — /ready, /livez, /katalog routes
		ws,            // 2. HTTPS webhook server — /validate, /mutate, /convert, etc.
		kube,          // 3. REST clients — already started, managed for Stop()
		ev,            // 4. event recorder — depends on kube
		queueRegistry, // 5. per-CRD bounded queues
		defaultWq,     // 6. default unbounded queue
		infFactory,    // 7. informer factory — starts watchers, closes ready channel
		kord,          // 8. dependency kordinator — starts workers in topo order
	}

	// ── 8. Orkestra ───────────────────────────────────────────────────────────
	// The supervisor. Calls Start() on each komponent in order.
	// On OS signal (SIGTERM/SIGINT) or fatal error, calls Stop() in reverse.
	// Graceful shutdown: drains queues before stopping workers.
	o := ork.NewOrkestra(
		kfg.Katalog().ShutdownGracePeriod,
		kfg.Ork().LogLevel,
	)
	o.Register(komponents)

	return &orkestraKfg{
		konfig:   kfg,
		katalog:  kat,
		komp:     &komponents,
		event:    ev,
		kube:     kube,
		kord:     kord,
		orkestra: o,
		tlsCert:  tlsCert,
		tlsKey:   tlsKey,
	}
}
