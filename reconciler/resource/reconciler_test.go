package resource

// func TestReconciler(t *testing.T) {
// 	type args struct {
// 		ctx     context.Context
// 		cl      client.Client
// 		s       *runtime.Scheme
// 		desired client.Object
// 		enabled bool
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		cfg     ReconcilerConfig
// 		want    client.Object
// 		wantErr bool
// 	}{
// 		{
// 			name: "Reconciles properties and applies mutations",
// 			args: args{
// 				ctx: context.TODO(),
// 				cl: fake.NewClientBuilder().WithObjects(
// 					&corev1.Service{
// 						TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 						ObjectMeta: metav1.ObjectMeta{
// 							Name:      "service",
// 							Namespace: "ns",
// 						},
// 						Spec: corev1.ServiceSpec{
// 							Type:                  corev1.ServiceTypeLoadBalancer,
// 							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
// 							SessionAffinity:       corev1.ServiceAffinityNone,
// 							Ports: []corev1.ServicePort{
// 								{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP},
// 								{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
// 							},
// 							Selector: map[string]string{"selector": "deployment"},
// 						},
// 					}).Build(),
// 				s: scheme.Scheme,
// 				desired: &corev1.Service{
// 					TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:        "service",
// 						Namespace:   "ns",
// 						Annotations: map[string]string{"key": "value"},
// 					},
// 					Spec: corev1.ServiceSpec{
// 						Type: corev1.ServiceTypeLoadBalancer,
// 						Ports: []corev1.ServicePort{{
// 							Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
// 					},
// 				},
// 				enabled: true,
// 			},
// 			cfg: ReconcilerConfig{
// 				ReconcileProperties: []Property{
// 					"metadata.annotations",
// 					"spec.ports",
// 					"spec.selector",
// 				},
// 				Mutations: []MutationFunction{
// 					func(ctx context.Context, cl client.Client, instance client.Object, desired client.Object) error {
// 						instance.(*corev1.Service).Spec.InternalTrafficPolicy = util.Pointer(corev1.ServiceInternalTrafficPolicyLocal)
// 						return nil
// 					},
// 				},
// 			},
// 			want: &corev1.Service{
// 				TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:        "service",
// 					Namespace:   "ns",
// 					Annotations: map[string]string{"key": "value"},
// 				},
// 				Spec: corev1.ServiceSpec{
// 					Type:                  corev1.ServiceTypeLoadBalancer,
// 					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
// 					InternalTrafficPolicy: util.Pointer(corev1.ServiceInternalTrafficPolicyLocal),
// 					SessionAffinity:       corev1.ServiceAffinityNone,
// 					Ports: []corev1.ServicePort{{
// 						Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
// 				},
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "Ignores properties",
// 			args: args{
// 				ctx: context.TODO(),
// 				cl: fake.NewClientBuilder().WithObjects(
// 					&corev1.Service{
// 						TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 						ObjectMeta: metav1.ObjectMeta{
// 							Name:      "service",
// 							Namespace: "ns",
// 						},
// 						Spec: corev1.ServiceSpec{
// 							Type:                  corev1.ServiceTypeLoadBalancer,
// 							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
// 							SessionAffinity:       corev1.ServiceAffinityNone,
// 							Ports: []corev1.ServicePort{
// 								{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33000},
// 								{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33001},
// 							},
// 							Selector: map[string]string{"selector": "deployment"},
// 						},
// 					}).Build(),
// 				s: scheme.Scheme,
// 				desired: &corev1.Service{
// 					TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:      "service",
// 						Namespace: "ns",
// 					},
// 					Spec: corev1.ServiceSpec{
// 						Type: corev1.ServiceTypeLoadBalancer,
// 						Ports: []corev1.ServicePort{
// 							{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP},
// 							{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
// 						},
// 					}},
// 				enabled: true,
// 			},
// 			cfg: ReconcilerConfig{
// 				ReconcileProperties: []Property{"spec.ports"},
// 				IgnoreProperties:    []Property{"spec.ports[*].nodePort"},
// 				Mutations:           []MutationFunction{},
// 			},
// 			want: &corev1.Service{
// 				TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "service",
// 					Namespace: "ns",
// 				},
// 				Spec: corev1.ServiceSpec{
// 					Type:                  corev1.ServiceTypeLoadBalancer,
// 					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
// 					SessionAffinity:       corev1.ServiceAffinityNone,
// 					Ports: []corev1.ServicePort{
// 						{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33000},
// 						{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33001},
// 					},
// 					Selector: map[string]string{"selector": "deployment"},
// 				}},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := tt.cfg.Reconcile(tt.args.ctx, tt.args.cl, tt.args.s, tt.args.desired,
// 				tt.args.enabled); (err != nil) != tt.wantErr {
// 				t.Errorf("Reconciler() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			got := tt.want.DeepCopyObject().(client.Object)
// 			tt.args.cl.Get(context.TODO(), util.ObjectKey(tt.want), got)
// 			if diff := cmp.Diff(got, tt.want, util.IgnoreProperty("ResourceVersion")); len(diff) > 0 {
// 				t.Errorf("Reconciler() diff in set.current = %v", diff)
// 			}
// 		})
// 	}
// }

// func TestServiceTemplate_ResourceReconciler(t *testing.T) {
// 	type fields struct {
// 		Template  func() *corev1.Service
// 		IsEnabled bool
// 	}
// 	type args struct {
// 		ctx     context.Context
// 		cl      client.Client
// 		desired client.Object
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    *corev1.Service
// 		wantErr bool
// 	}{
// 		{
// 			name: "",
// 			fields: fields{
// 				Template:  nil,
// 				IsEnabled: true,
// 			},
// 			args: args{
// 				ctx: context.TODO(),
// 				cl: fake.NewClientBuilder().WithObjects(
// 					&corev1.Service{
// 						ObjectMeta: metav1.ObjectMeta{
// 							Name:      "service",
// 							Namespace: "default",
// 						},
// 						Spec: corev1.ServiceSpec{
// 							Type:                  corev1.ServiceTypeLoadBalancer,
// 							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
// 							SessionAffinity:       corev1.ServiceAffinityNone,
// 							Ports: []corev1.ServicePort{
// 								{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33000},
// 								{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
// 							},
// 							Selector: map[string]string{"selector": "deployment"},
// 						},
// 					},
// 				).Build(),
// 				desired: &corev1.Service{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:        "service",
// 						Namespace:   "default",
// 						Annotations: map[string]string{"key": "value"},
// 					},
// 					Spec: corev1.ServiceSpec{
// 						Type: corev1.ServiceTypeLoadBalancer,
// 						Ports: []corev1.ServicePort{{
// 							Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
// 						Selector: map[string]string{"selector": "deployment"},
// 					},
// 				},
// 			},
// 			want: &corev1.Service{
// 				TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:        "service",
// 					Namespace:   "default",
// 					Annotations: map[string]string{"key": "value"},
// 				},
// 				Spec: corev1.ServiceSpec{
// 					Type:                  corev1.ServiceTypeLoadBalancer,
// 					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
// 					SessionAffinity:       corev1.ServiceAffinityNone,
// 					Ports: []corev1.ServicePort{{
// 						Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33000}},
// 					Selector: map[string]string{"selector": "deployment"},
// 				},
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			st := ServiceTemplate{
// 				Template:  tt.fields.Template,
// 				IsEnabled: tt.fields.IsEnabled,
// 			}
// 			if err := st.ResourceReconciler(tt.args.ctx, tt.args.cl, tt.args.desired); (err != nil) != tt.wantErr {
// 				t.Errorf("ServiceTemplate.ResourceReconciler() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			got := &corev1.Service{}
// 			tt.args.cl.Get(context.TODO(), util.ObjectKey(tt.want), got)
// 			if diff := cmp.Diff(got, tt.want, ignoreResourceVersion()); len(diff) > 0 {
// 				t.Errorf("ChangeSet_removeMatchingProperties() diff in set.current = %v", diff)
// 			}
// 		})
// 	}
// }

// func ignoreResourceVersion() cmp.Option {
// 	return cmp.FilterPath(
// 		func(path cmp.Path) bool {
// 			if field, ok := path.Last().(cmp.StructField); ok {
// 				return strings.HasPrefix(field.Name(), "ResourceVersion")
// 			}
// 			return false
// 		},
// 		cmp.Ignore(),
// 	)
// }
