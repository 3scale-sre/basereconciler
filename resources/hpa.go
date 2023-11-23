package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/property"
	"github.com/3scale-ops/basereconciler/reconciler"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = HorizontalPodAutoscalerTemplate{}

// HorizontalPodAutoscalerTemplate has methods to generate and reconcile a HorizontalPodAutoscaler
type HorizontalPodAutoscalerTemplate struct {
	Template  func() *autoscalingv2.HorizontalPodAutoscaler
	IsEnabled bool
}

// Build returns a HorizontalPodAutoscaler resource
func (hpat HorizontalPodAutoscalerTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return hpat.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (hpat HorizontalPodAutoscalerTemplate) Enabled() bool {
	return hpat.IsEnabled
}

// ResourceReconciler implements a generic reconciler for HorizontalPodAutoscaler resources
func (hpat HorizontalPodAutoscalerTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "HorizontalPodAutoscaler", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*autoscalingv2.HorizontalPodAutoscaler)

	instance := &autoscalingv2.HorizontalPodAutoscaler{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if hpat.Enabled() {
				err = cl.Create(ctx, desired)
				if err != nil {
					return fmt.Errorf("unable to create object: " + err.Error())
				}
				logger.Info("resource created")
				return nil

			} else {
				return nil
			}
		}

		return err
	}

	/* Delete and return if not enabled */
	if !hpat.Enabled() {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return fmt.Errorf("unable to delete object: " + err.Error())
		}
		logger.Info("resource deleted")
		return nil
	}

	/* Ensure the resource is in its desired state */
	needsUpdate = property.EnsureDesired(logger,
		property.NewChangeSet[map[string]string]("metadata.labels", &instance.ObjectMeta.Labels, &desired.ObjectMeta.Labels),
		property.NewChangeSet[map[string]string]("metadata.annotations", &instance.ObjectMeta.Annotations, &desired.ObjectMeta.Annotations),
		property.NewChangeSet[autoscalingv2.CrossVersionObjectReference]("spec.scaleTargetRef", &instance.Spec.ScaleTargetRef, &desired.Spec.ScaleTargetRef),
		property.NewChangeSet[int32]("spec.minReplicas", instance.Spec.MinReplicas, desired.Spec.MinReplicas),
		property.NewChangeSet[int32]("spec.maxReplicas", &instance.Spec.MaxReplicas, &desired.Spec.MaxReplicas),
		property.NewChangeSet[[]autoscalingv2.MetricSpec]("spec.metrics", &instance.Spec.Metrics, &desired.Spec.Metrics),
	)

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("resource updated")
	}

	return nil
}
