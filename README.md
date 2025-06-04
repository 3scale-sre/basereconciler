# basereconciler

Basereconciler is an attempt to create a reconciler that can be imported and used in any controller-runtime based controller to perform the most common tasks a controller usually performs. It's a bunch of code that it's typically written again and again for every and each controller and that can be abstracted to work in a more generic way to avoid the repetition and improve code mantainability.
At the moment basereconciler can perform the following tasks:

* **Get the custom resource and perform some common tasks on it**:
  * Management of initialization logic: custom initialization functions can be passed to perform initialization tasks on the custom resource. Initialization can be done persisting changes in the API server (use reconciler.WithInitializationFunc) or without persisting them (reconciler.WithInMemoryInitializationFunc).
  * Management of resource finalizer: some custom resources required more complex finalization logic. For this to happen a finalizer must be in place. Basereconciler can keep this finalizer in place and remove it when necessary during resource finalization.
  * Management of finalization logic: it checks if the resource is being finalized and executed the finalization logic passed to it if that is the case. When all finalization logic is completed it removes the finalizer on the custom resource.
* **Reconcile resources owned by the custom resource**: basereconciler can keep the owned resources of a custom resource in it's desired state. It works for any resource type, and only requires that the user configures how each specific resource type has to be configured. The resource reconciler only works in "update mode" right now, so any operation to transition a given resource from its live state to its desired state will be an Update. We might add a "patch mode" in the future.
* **Reconcile custom resource status**: if the custom resource implements a certain interface, basereconciler can also be in charge of reconciling the status.
* **Resource pruner**: when the reconciler stops seeing a certain resource, owned by the custom resource, it will prune them as it understands that the resource is no longer required. The resource pruner can be disabled globally or enabled/disabled on a per resource basis based on an annotation.

## Basic Usage

The following example is a kubebuilder bootstrapped controller that uses basereconciler to reconcile several resources owned by a custom resource. Explanations inline in the code.

```go
package controllers

import (
	"context"

	"github.com/3scale-sre/basereconciler/config"
	"github.com/3scale-sre/basereconciler/mutators"
	"github.com/3scale-sre/basereconciler/reconciler"
	"github.com/3scale-sre/basereconciler/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	webappv1 "my.domain/guestbook/api/v1"
)

// Use the init function to configure the behavior of the controller. In this case we use
// "SetDefaultReconcileConfigForGVK" to specify the paths that need to be reconciled/ignored
// for each resource type. Check the "github.com/3scale-sre/basereconciler/config" for more
// configuration options
func init() {
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("apps/v1", "Deployment"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec",
			},
			IgnoreProperties: []string{
				"metadata.annotations['deployment.kubernetes.io/revision']",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		// specifying a config for an empty GVK will
		// set a default fallback config for any gvk that is not
		// explicitely declared in the configuration. Think of it
		// as a wildcard.
		schema.GroupVersionKind{},
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec",
			},
		})
}

// GuestbookReconciler reconciles a Guestbook object
type GuestbookReconciler struct {
	*reconciler.Reconciler
}

// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestbooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestbooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",namespace=placeholder,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *GuestbookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// configure the logger for the controller. The function also returns a modified
	// copy of the context that includes the logger so it's easily passed around to other functions.
	ctx, logger := r.Logger(ctx, "guestbook", req.NamespacedName)

	// ManageResourceLifecycle will take care of retrieving the custom resoure from the API. It is also in charge of the resource
	// lifecycle: initialization and finalization logic. In this example, we are configuring a finalizer in our custom resource and passing
	// a finalization function that will casuse a log line to show when the resource is being deleted.
	guestbook := &webappv1.Guestbook{}
	result := r.ManageResourceLifecycle(ctx, req, guestbook,
		reconciler.WithInitializationFunc(
			func(context.Context, client.Client) error {
				logger.Info("initializing resource")
				return nil
			}),
		reconciler.WithFinalizer("guestbook-finalizer"),
		reconciler.WithFinalizationFunc(
			func(context.Context, client.Client) error {
				logger.Info("finalizing resource")
				return nil
			}),
	)
	if result.ShouldReturn() {
		return result.Values()
	}

	// ReconcileOwnedResources creates/updates/deletes the resoures that our custom resource owns.
	// It is a list of templates, in this case generated from the base of an object we provide.
	// Modifiers can be added to the template to get live values from the k8s API, like in this example
	// with the Service. Check the documentation of the "github.com/3scale-sre/basereconciler/resource"
	// for more information on building templates.
	result = r.ReconcileOwnedResources(ctx, guestbook, []resource.TemplateInterface{

		resource.NewTemplateFromObjectFunction[*appsv1.Deployment](
			func() *appsv1.Deployment {
				return &appsv1.Deployment{
					// define your object here
				}
			}),

		resource.NewTemplateFromObjectFunction[*corev1.Service](
			func() *corev1.Service {
				return &corev1.Service{
					// define your object here
				}
			}).
			// Retrieve the live values that kube-controller-manager sets
			// in the Service spec to avoid overwrting them
			WithMutation(mutators.SetServiceLiveValues()).
			// There are some useful mutations in the "github.com/3scale-sre/basereconciler/mutators"
			// package or you can pass your own mutation functions
			WithMutation(func(ctx context.Context, cl client.Client, desired client.Object) error {
				// your mutation logic here
				return nil
			}).
			// The templates are reconciled using the global config defined in the init() function
			// but in this case we are passing a custom config that will apply
			// only to the reconciliation of this template
			WithEnsureProperties([]resource.Property{"spec"}).
			WithIgnoreProperties([]resource.Property{"spec.clusterIP", "spec.clusterIPs"}),
	})

	if result.ShouldReturn() {
		return result.Values()
	}

	return ctrl.Result{}, nil
}

func (r *GuestbookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// SetupWithDynamicTypeWatches will configure the controller to dynamically
	// watch for any resource type that the controller owns.
	return reconciler.SetupWithDynamicTypeWatches(r,
		ctrl.NewControllerManagedBy(mgr).
			For(&webappv1.Guestbook{}),
	)
}
```

Then you just need to register the controller with the controller-runtime manager and you are all set!

```go
[...]
	if err = (&controllers.GuestbookReconciler{
		Reconciler: reconciler.NewFromManager(mgr).WithLogger(ctrl.Log.WithName("controllers").WithName("Guestbook")),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Guestbook")
		os.Exit(1)
	}
[...]
```

## Status reconciliation

The status reconciliation feature works for custom resources that deploy workloads. Right now only owned Deployments and StatefulSets are supported.

To use the status reconciliation feature add this to your controller, indicating the Deployments and StatefulSets that are owned by the custom resource:

```go
[...]
	result = r.ReconcileStatus(ctx, instance,
		[]types.NamespacedName{{Name: "my-deployment", Namespace: "ns"}},
		[]types.NamespacedName{{Name: "my-statefulset", Namespace: "ns"}},
	)
	if result.ShouldReturn() {
		return result.Values()
	}
[...]
```

The status reconciliation is modular, with 3 tiers of functionality depending on the interfaces that the custom resource implements:

* **Tier 1**. The custom resource has to implement the [ObjectWithAppStatus interface](reconciler/status.go#L16) and the status object has to implement the [AppStatus interface](reconciler/status.go#L26). Implementing these interfaces will make the controller  lookup the current status of the Deployment/s and/or StatefulSet/s and copy over their status/es to the custom resource's status.

* **Tier 2**. Additionally to the interfaces described in **Tier 1**, also the [AppStatusWithHealth interface](reconciler/status.go#L35) is implemented. This will make the controller calculate the health of each Deployment/StatefulSet and write it down to the custom resource status.

* **Tier 3**. For workloads composed by more that 1 Deployment/StatefulSet, if the [AppStatusWithAggregatedHealth interface](reconciler/status.go#L46) is also implemented, the controller will calculate the overall health of the custom resources based on each individual Deployment/StatefulSet health status and then written to the status of the custom resource. The overall health is always the worse health status found across all owned Deployments/StatefulSets.

An example of a custom resource that implements **Tier 1** status reconciliation can be found in [this test](test/api/v1alpha1/test_types.go). To implement the other 2 tiers just add the required methods to the status object.

There is also the option to pass custom status reconciliation code in case the custom resource has status fields that require specific logic. An example of this could be:

```go
[...]
	result = r.ReconcileStatus(ctx, instance,
		[]types.NamespacedName{{Name: "my-deployment", Namespace: "ns"}},
		[]types.NamespacedName{{Name: "my-statefulset", Namespace: "ns"}},
		func() (bool, error) {
			if condition {
				instance.Status.MyCustomField = value
				// return true when the field/s requires update
				return true, nil
			}
			// return false when the field/s doesn't/don't require update
			return false, nil
		})
	)
	if result.ShouldReturn() {
		return result.Values()
	}
[...]
```