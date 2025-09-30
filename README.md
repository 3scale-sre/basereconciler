# basereconciler

**basereconciler** is a comprehensive reconciliation library designed to be imported and used in any controller-runtime based controller to perform the most common tasks a controller typically performs. It abstracts repetitive reconciliation code into a reusable, generic framework that improves code maintainability and reduces boilerplate.

## Core Features

* **Resource Lifecycle Management**:
  * **Initialization Logic**: Custom initialization functions can be passed to perform initialization tasks on the custom resource. Initialization can be done persisting changes in the API server (use `reconciler.WithInitializationFunc`) or without persisting them (use `reconciler.WithInMemoryInitializationFunc`).
  * **Finalizer Management**: Some custom resources require more complex finalization logic. For this to happen, a finalizer must be in place. basereconciler can keep this finalizer in place and remove it when necessary during resource finalization.
  * **Finalization Logic**: It checks if the resource is being finalized and executes the finalization logic passed to it. When all finalization logic is completed, it removes the finalizer from the custom resource.

* **Intelligent Resource Reconciliation**: basereconciler can keep the owned resources of a custom resource in their desired state. It works for any resource type with configurable reconciliation behavior:
  * **Dual Operation Modes**: Supports both Update operations (full object replacement) and Patch operations (strategic merge patches for efficiency).
  * **Property-based Control**: Fine-grained control over which fields are reconciled via JSONPath expressions.
  * **Automatic GVK Inference**: Templates automatically determine their resource type from generic type parameters.
  * **Template-driven Configuration**: Operation type (Update/Patch) and reconciliation properties configured per template.
  * **Global Configuration Support**: Default configurations can be set globally with per-template overrides.

* **Status Reconciliation**: If the custom resource implements a certain interface, basereconciler can also be in charge of reconciling the status.

* **Resource Pruner**: When the reconciler stops seeing a certain resource owned by the custom resource, it will prune them as it understands that the resource is no longer required. The resource pruner can be disabled globally or enabled/disabled on a per-resource basis based on an annotation.

## Basic Usage

The following example shows a kubebuilder-bootstrapped controller that uses basereconciler to reconcile several resources owned by a custom resource. Explanations are inline in the code.

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

// Use the init function to configure the global behavior of the controller. This is optional -
// templates can also specify their own reconciliation configuration via WithEnsureProperties/WithIgnoreProperties.
// Check the "github.com/3scale-sre/basereconciler/config" package for more configuration options.
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
		// Specifying a config for an empty GVK will
		// set a default fallback config for any GVK that is not
		// explicitly declared in the configuration. Think of it
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
	// Configure the logger for the controller. The function also returns a modified
	// copy of the context that includes the logger so it's easily passed around to other functions.
	ctx, logger := r.Logger(ctx, "guestbook", req.NamespacedName)

	// ManageResourceLifecycle will take care of retrieving the custom resource from the API. It is also in charge of the resource
	// lifecycle: initialization and finalization logic. In this example, we are configuring a finalizer in our custom resource and passing
	// a finalization function that will cause a log line to show when the resource is being deleted.
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

	// ReconcileOwnedResources creates/updates/deletes the resources that our custom resource owns.
	// It takes a list of templates that automatically infer their GVK from the generic type parameter.
	// Templates can be configured with different operation modes (Update/Patch) and property configurations.
	// Check the documentation of the "github.com/3scale-sre/basereconciler/resource" package for more information.
	result = r.ReconcileOwnedResources(ctx, guestbook, []resource.TemplateInterface{

		// Basic template using automatic GVK inference and default global configuration
		resource.NewTemplateFromObjectFunction(func() *appsv1.Deployment {
			return &appsv1.Deployment{
				// define your deployment here
			}
		}),

		// Advanced template with mutations, custom reconciliation configuration, and patch operation
		resource.NewTemplateFromObjectFunction(func() *corev1.Service {
			return &corev1.Service{
				// define your service here  
			}
		}).
			// Retrieve the live values that kube-controller-manager sets
			// in the Service spec to avoid overwriting them
			WithMutation(mutators.SetServiceLiveValues()).
			// There are useful mutations in the "github.com/3scale-sre/basereconciler/mutators"
			// package, or you can pass your own mutation functions
			WithMutation(func(ctx context.Context, cl client.Client, desired client.Object) error {
				// your custom mutation logic here
				return nil
			}).
			// Configure which properties to reconcile (overrides global configuration for this template)
			WithEnsureProperties([]resource.Property{"spec"}).
			WithIgnoreProperties([]resource.Property{"spec.clusterIP", "spec.clusterIPs"}).
			// Use patch operation for better performance and reduced conflicts
			WithModifyOp(resource.ModifyOpPatch),
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

Then you just need to register the controller with the controller-runtime manager and you're all set!

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

## Advanced Features

### Operation Modes

The resource package supports two reconciliation modes:

- **Update Mode** (`ModifyOpUpdate`): Performs full object replacement using `client.Update()`. This is the default mode for backward compatibility.
- **Patch Mode** (`ModifyOpPatch`): Uses strategic merge patches via `client.Patch()` with `client.MergeFrom()`. This is more efficient and reduces conflicts.

Configure operation mode per template:

```go
template := resource.NewTemplate(builder).
    WithModifyOp(resource.ModifyOpPatch)  // Use patch mode for better performance
```

### Automatic GVK Inference

Templates automatically determine their GroupVersionKind from the generic type parameter, eliminating boilerplate:

```go
// Automatically infers GVK as apps/v1 Deployment
template := resource.NewTemplateFromObjectFunction(func() *appsv1.Deployment {
    return &appsv1.Deployment{...}
})
```

### Default Scheme Management

Set a global default scheme to avoid passing it to every constructor:

```go
import myscheme "github.com/myorg/myoperator/pkg/scheme"

func init() {
    resource.Scheme = myscheme.Scheme  // Now used by default for all templates
}
```

### Property-based Reconciliation

Fine-grained control over which fields are reconciled using JSONPath expressions:

```go
template := resource.NewTemplate(builder).
    // Only reconcile these specific fields
    WithEnsureProperties([]resource.Property{
        "metadata.labels",
        "spec.replicas", 
        "spec.template.spec.containers[0].image",
    }).
    // Ignore these fields even if they're within an ensured path
    WithIgnoreProperties([]resource.Property{
        "spec.clusterIP",                                               // Let Kubernetes manage
        "metadata.annotations['deployment.kubernetes.io/revision']",    // Let deployment controller manage
    })
```

## Status Reconciliation

The status reconciliation feature works for custom resources that deploy workloads. Currently, only owned Deployments and StatefulSets are supported.

To use the status reconciliation feature, add this to your controller, indicating the Deployments and StatefulSets that are owned by the custom resource:

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

The status reconciliation is modular, with three tiers of functionality depending on the interfaces that the custom resource implements:

* **Tier 1**: The custom resource must implement the [ObjectWithAppStatus interface](reconciler/status.go#L16) and the status object must implement the [AppStatus interface](reconciler/status.go#L26). Implementing these interfaces will make the controller look up the current status of the Deployment(s) and/or StatefulSet(s) and copy over their statuses to the custom resource's status.

* **Tier 2**: In addition to the interfaces described in **Tier 1**, the [AppStatusWithHealth interface](reconciler/status.go#L35) is also implemented. This will make the controller calculate the health of each Deployment/StatefulSet and write it to the custom resource status.

* **Tier 3**: For workloads composed of more than one Deployment/StatefulSet, if the [AppStatusWithAggregatedHealth interface](reconciler/status.go#L46) is also implemented, the controller will calculate the overall health of the custom resource based on each individual Deployment/StatefulSet health status and then write it to the status of the custom resource. The overall health is always the worst health status found across all owned Deployments/StatefulSets.

An example of a custom resource that implements **Tier 1** status reconciliation can be found in [this test](test/api/v1alpha1/test_types.go). To implement the other two tiers, just add the required methods to the status object.

There is also the option to pass custom status reconciliation code in case the custom resource has status fields that require specific logic. An example of this would be:

```go
[...]
	result = r.ReconcileStatus(ctx, instance,
		[]types.NamespacedName{{Name: "my-deployment", Namespace: "ns"}},
		[]types.NamespacedName{{Name: "my-statefulset", Namespace: "ns"}},
		func() (bool, error) {
			if condition {
				instance.Status.MyCustomField = value
				// Return true when the field requires update
				return true, nil
			}
			// Return false when the field doesn't require update
			return false, nil
		})
	)
	if result.ShouldReturn() {
		return result.Values()
	}
[...]
```