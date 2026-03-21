package reconciler

// var err error

// switch {
// case r.hooks.OnReconcile != nil:
// 	err = r.hooks.OnReconcile(ctx, obj)

// case r.rc.OnCreate != nil || r.rc.OnReconcile != nil:
// 	err = r.runTemplateReconcile(ctx, obj)

// default:
// 	logger.FromContext(ctx).Debug().
// 		Str("name", obj.GetName()).
// 		Msg("reconciled (no-op)")
// 	return nil
// }

// if err != nil {
// 	logger.FromContext(ctx).Error().Err(err).
// 		Str("name", obj.GetName()).
// 		Msgf("reconciliation failed for %s", r.crd.GVK)
// 	r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"ReconcileError",
// 		fmt.Sprintf("Failed to reconcile %s %s/%s: %v",
// 			r.crd.GVK, obj.GetNamespace(), obj.GetName(), err))
// 	return err
// }

// r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.Kind+"Reconciled",
// 	fmt.Sprintf("Successfully reconciled %s %s/%s",
// 		r.crd.GVK, obj.GetNamespace(), obj.GetName()))

// return nil
