package resource

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTemplate_Apply(t *testing.T) {

	podTemplate := &Template[*corev1.Pod]{
		TemplateBuilder: func(client.Object) (*corev1.Pod, error) {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns"},
			}, nil
		},
	}

	podTemplate.Apply(func(o client.Object) (*corev1.Pod, error) {
		o.SetAnnotations(map[string]string{"key": "value"})
		return o.(*corev1.Pod), nil
	})

	got, _ := podTemplate.Build(context.TODO(), fake.NewClientBuilder().Build(), nil)
	want := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns", Annotations: map[string]string{"key": "value"}},
	}

	if diff := cmp.Diff(got, want); len(diff) > 0 {
		t.Errorf("(Template).Apply() diff = %v", diff)
	}
}

func Test_NewTemplate(t *testing.T) {
	scheme := GetScheme()

	builderFunc := func(owner client.Object) (*corev1.Pod, error) {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}, nil
	}

	template := NewTemplate(builderFunc, scheme)

	// Test that template is properly initialized
	if template == nil {
		t.Fatal("NewTemplate returned nil")
	}

	// Test that IsEnabled defaults to true
	if !template.IsEnabled {
		t.Error("Expected IsEnabled to default to true")
	}

	// Test that TemplateBuilder is set
	if template.TemplateBuilder == nil {
		t.Error("Expected TemplateBuilder to be set")
	}

	// Test that GVK is correctly inferred
	expectedGVK := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}
	if template.GVK != expectedGVK {
		t.Errorf("Expected GVK %v, got %v", expectedGVK, template.GVK)
	}

	// Test that TemplateBuilder function works
	pod, err := template.TemplateBuilder(nil)
	if err != nil {
		t.Errorf("TemplateBuilder failed: %v", err)
	}
	if pod.Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got %s", pod.Name)
	}
}

func Test_NewTemplateFromObjectFunction(t *testing.T) {
	scheme := GetScheme()

	objectFunc := func() *corev1.Service {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: 80, Protocol: corev1.ProtocolTCP},
				},
			},
		}
	}

	template := NewTemplateFromObjectFunction(objectFunc, scheme)

	// Test that template is properly initialized
	if template == nil {
		t.Fatal("NewTemplateFromObjectFunction returned nil")
	}

	// Test that IsEnabled defaults to true
	if !template.IsEnabled {
		t.Error("Expected IsEnabled to default to true")
	}

	// Test that TemplateBuilder is set
	if template.TemplateBuilder == nil {
		t.Error("Expected TemplateBuilder to be set")
	}

	// Test that GVK is correctly inferred
	expectedGVK := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}
	if template.GVK != expectedGVK {
		t.Errorf("Expected GVK %v, got %v", expectedGVK, template.GVK)
	}

	// Test that TemplateBuilder function works and ignores input parameter (used for chaining)
	service, err := template.TemplateBuilder(&corev1.Pod{}) // Pass any input, should be ignored by this simple function
	if err != nil {
		t.Errorf("TemplateBuilder failed: %v", err)
	}
	if service.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got %s", service.Name)
	}
	if len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Port != 80 {
		t.Error("Expected service to have port 80")
	}
}

func Test_GetReconcileOptions(t *testing.T) {
	tests := []struct {
		name     string
		template TemplateInterface
		want     ReconcileOptions
	}{
		{
			name:     "Returns default config",
			template: &Template[*corev1.Pod]{},
			want: ReconcileOptions{
				EnsureProperties: []Property{"metadata.annotations", "metadata.labels", "spec"},
				IgnoreProperties: []Property{},
				ModifyOp:         ModifyOpUpdate,
			},
		},
		{
			name: "Returns explicit config",
			template: &Template[*corev1.Pod]{
				EnsureProperties: []Property{"a.b.c"},
				IgnoreProperties: []Property{"x"},
			},
			want: ReconcileOptions{
				EnsureProperties: []Property{"a.b.c"},
				IgnoreProperties: []Property{"x"},
				ModifyOp:         ModifyOpUpdate,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.GetReconcileOptions()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetReconcileOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Template_Build(t *testing.T) {
	scheme := GetScheme()
	ctx := context.Background()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name             string
		template         *Template[*corev1.ConfigMap]
		wantName         string
		wantMutationCall bool
		wantErr          bool
	}{
		{
			name: "Build without mutations",
			template: &Template[*corev1.ConfigMap]{
				TemplateBuilder: func(o client.Object) (*corev1.ConfigMap, error) {
					return &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{"key": "value"},
					}, nil
				},
			},
			wantName:         "test-configmap",
			wantMutationCall: false,
			wantErr:          false,
		},
		{
			name: "Build with mutations",
			template: &Template[*corev1.ConfigMap]{
				TemplateBuilder: func(o client.Object) (*corev1.ConfigMap, error) {
					return &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{"key": "value"},
					}, nil
				},
				TemplateMutations: []TemplateMutationFunction{
					func(ctx context.Context, cl client.Client, obj client.Object) error {
						cm := obj.(*corev1.ConfigMap)
						cm.Data["mutated"] = "true"
						return nil
					},
				},
			},
			wantName:         "test-configmap",
			wantMutationCall: true,
			wantErr:          false,
		},
		{
			name: "Build with failing builder",
			template: &Template[*corev1.ConfigMap]{
				TemplateBuilder: func(o client.Object) (*corev1.ConfigMap, error) {
					return nil, fmt.Errorf("builder error")
				},
			},
			wantErr: true,
		},
		{
			name: "Build with failing mutation",
			template: &Template[*corev1.ConfigMap]{
				TemplateBuilder: func(o client.Object) (*corev1.ConfigMap, error) {
					return &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
					}, nil
				},
				TemplateMutations: []TemplateMutationFunction{
					func(ctx context.Context, cl client.Client, obj client.Object) error {
						return fmt.Errorf("mutation error")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.template.Build(ctx, cl, nil)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Build() expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Build() unexpected error: %v", err)
				return
			}

			if obj == nil {
				t.Error("Build() returned nil object")
				return
			}

			cm := obj.(*corev1.ConfigMap)
			if cm.Name != tt.wantName {
				t.Errorf("Build() object name = %v, want %v", cm.Name, tt.wantName)
			}

			if tt.wantMutationCall {
				if cm.Data["mutated"] != "true" {
					t.Error("Build() mutation was not applied")
				}
			}

			// Test that result is a deep copy (modifying it shouldn't affect template)
			cm.Name = "modified"
			obj2, _ := tt.template.Build(ctx, cl, nil)
			if obj2.(*corev1.ConfigMap).Name == "modified" {
				t.Error("Build() did not return a deep copy")
			}
		})
	}
}

func Test_Template_Apply(t *testing.T) {
	scheme := GetScheme()

	// Create a base template
	baseTemplate := NewTemplate(func(owner client.Object) (*corev1.Service, error) {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "base-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{Port: 80}},
			},
		}, nil
	}, scheme)

	// Apply a transformation
	transformation := func(input client.Object) (*corev1.Service, error) {
		svc := input.(*corev1.Service)
		// Modify the service
		svc.Name = "transformed-service"
		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{Port: 443})
		svc.Labels = map[string]string{"transformed": "true"}
		return svc, nil
	}

	transformedTemplate := baseTemplate.Apply(transformation)

	// Test that Apply returns the same template instance (fluent API)
	if transformedTemplate != baseTemplate {
		t.Error("Apply() should return the same template instance for chaining")
	}

	// Test that the transformation is applied when building
	ctx := context.Background()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	obj, err := transformedTemplate.Build(ctx, client, nil)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	svc := obj.(*corev1.Service)

	// Verify transformations were applied
	if svc.Name != "transformed-service" {
		t.Errorf("Expected name 'transformed-service', got %s", svc.Name)
	}

	if len(svc.Spec.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(svc.Spec.Ports))
	}

	if svc.Labels["transformed"] != "true" {
		t.Error("Expected 'transformed' label to be 'true'")
	}
}

func Test_Template_Apply_WithError(t *testing.T) {
	scheme := GetScheme()

	baseTemplate := NewTemplate(func(owner client.Object) (*corev1.Service, error) {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "base-service"},
		}, nil
	}, scheme)

	// Apply a transformation that fails
	failingTransformation := func(input client.Object) (*corev1.Service, error) {
		return nil, fmt.Errorf("transformation failed")
	}

	transformedTemplate := baseTemplate.Apply(failingTransformation)

	// Test that the error is propagated
	ctx := context.Background()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := transformedTemplate.Build(ctx, client, nil)
	if err == nil {
		t.Error("Expected Build() to fail when transformation fails")
	}

	if err.Error() != "transformation failed" {
		t.Errorf("Expected error 'transformation failed', got %v", err)
	}
}

func Test_Template_Apply_ChainMultiple(t *testing.T) {
	scheme := GetScheme()

	// Test chaining multiple Apply calls
	template := NewTemplate(func(owner client.Object) (*corev1.Service, error) {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "base",
				Namespace: "default",
			},
		}, nil
	}, scheme).
		Apply(func(input client.Object) (*corev1.Service, error) {
			svc := input.(*corev1.Service)
			svc.Name = "step1"
			return svc, nil
		}).
		Apply(func(input client.Object) (*corev1.Service, error) {
			svc := input.(*corev1.Service)
			svc.Name = "step2"
			svc.Labels = map[string]string{"step": "2"}
			return svc, nil
		})

	ctx := context.Background()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	obj, err := template.Build(ctx, client, nil)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	svc := obj.(*corev1.Service)

	// Verify all transformations were applied in sequence
	if svc.Name != "step2" {
		t.Errorf("Expected final name 'step2', got %s", svc.Name)
	}

	if svc.Labels["step"] != "2" {
		t.Error("Expected 'step' label to be '2'")
	}
}

func Test_Template_GetGVK(t *testing.T) {
	expectedGVK := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}

	template := &Template[*corev1.Service]{
		GVK: expectedGVK,
	}

	got := template.GetGVK()
	if got != expectedGVK {
		t.Errorf("GetGVK() = %v, want %v", got, expectedGVK)
	}
}

func Test_inferGVKFromType(t *testing.T) {
	scheme := GetScheme()

	tests := []struct {
		name   string
		testFn func() schema.GroupVersionKind
		want   schema.GroupVersionKind
	}{
		{
			name: "Pod type",
			testFn: func() schema.GroupVersionKind {
				return inferGVKFromType[*corev1.Pod](scheme)
			},
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
		},
		{
			name: "Service type",
			testFn: func() schema.GroupVersionKind {
				return inferGVKFromType[*corev1.Service](scheme)
			},
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Service",
			},
		},
		{
			name: "ConfigMap type",
			testFn: func() schema.GroupVersionKind {
				return inferGVKFromType[*corev1.ConfigMap](scheme)
			},
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.testFn()
			if got != tt.want {
				t.Errorf("inferGVKFromType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_inferGVKFromType_PanicsOnUnregisteredType(t *testing.T) {
	scheme := GetScheme() // This won't have UnregisteredResource

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected inferGVKFromType to panic for unregistered type, but it didn't")
		}
	}()

	// This should panic because UnregisteredResource is not in the scheme
	inferGVKFromType[*UnregisteredResource](scheme)
}

func Test_NewTemplate_PanicsOnUnregisteredType(t *testing.T) {
	scheme := GetScheme() // This won't have UnregisteredResource

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected NewTemplate to panic for unregistered type, but it didn't")
		}
	}()

	// This should panic because UnregisteredResource is not in the scheme
	NewTemplate(func(client.Object) (*UnregisteredResource, error) {
		return &UnregisteredResource{}, nil
	}, scheme)
}

func Test_NewTemplateFromObjectFunction_PanicsOnUnregisteredType(t *testing.T) {
	scheme := GetScheme() // This won't have UnregisteredResource

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected NewTemplateFromObjectFunction to panic for unregistered type, but it didn't")
		}
	}()

	// This should panic because UnregisteredResource is not in the scheme
	NewTemplateFromObjectFunction(func() *UnregisteredResource {
		return &UnregisteredResource{}
	}, scheme)
}

// UnregisteredResource is a custom type not registered in any scheme
type UnregisteredResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// Implement client.Object interface
func (r *UnregisteredResource) DeepCopyObject() runtime.Object {
	return r.DeepCopy()
}

func (r *UnregisteredResource) DeepCopy() *UnregisteredResource {
	if r == nil {
		return nil
	}
	out := new(UnregisteredResource)
	r.DeepCopyInto(out)
	return out
}

func (r *UnregisteredResource) DeepCopyInto(out *UnregisteredResource) {
	*out = *r
	out.TypeMeta = r.TypeMeta
	r.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}
