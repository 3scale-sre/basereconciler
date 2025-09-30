package resource

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

// Scheme is the default runtime scheme used for GVK inference when no explicit scheme is provided
// to template constructors (NewTemplate, NewTemplateFromObjectFunction) or other functions that
// require scheme information.
//
// By default, this is set to the standard Kubernetes scheme (scheme.Scheme from client-go) which
// includes all core Kubernetes types (Pod, Service, Deployment, etc.).
//
// Users can override this globally to include custom resources or use a different scheme:
//
//	import myscheme "github.com/myorg/myoperator/pkg/scheme"
//
//	func init() {
//		resource.Scheme = myscheme.Scheme
//	}
//
// This provides a convenient way to avoid passing the scheme parameter to every template constructor
// while still allowing per-call overrides when needed.
var Scheme *runtime.Scheme = scheme.Scheme

// GetScheme returns the appropriate runtime scheme to use for GVK inference and other operations.
// If explicit schemes are provided, it returns the first one. Otherwise, it returns the package
// default scheme (resource.Scheme).
//
// This function is used internally by template constructors and other functions that need scheme
// access, providing a consistent way to handle both explicit and default scheme scenarios.
//
// Examples:
//
//	// Uses the default scheme (resource.Scheme)
//	scheme := GetScheme()
//
//	// Uses the provided scheme, ignoring the default
//	scheme := GetScheme(myCustomScheme)
//
//	// Multiple schemes provided, uses the first one
//	scheme := GetScheme(scheme1, scheme2) // Returns scheme1
func GetScheme(s ...*runtime.Scheme) *runtime.Scheme {
	if len(s) > 0 {
		return s[0]
	}
	return Scheme
}
