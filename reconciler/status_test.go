package reconciler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ AppStatusWithAggregatedHealth = &testStatus{}

type testWorkloadStatus struct {
	HealthStatus     string
	HealthMessage    string
	DeploymentStatus *appsv1.DeploymentStatus
}

type testStatus struct {
	AggregatedHealth string
	Workloads        map[string]*testWorkloadStatus
	UnimplementedStatefulSetStatus
}

func (ts *testStatus) GetDeploymentStatus(key types.NamespacedName) *appsv1.DeploymentStatus {
	return ts.Workloads[key.Name].DeploymentStatus
}
func (ts *testStatus) SetDeploymentStatus(key types.NamespacedName, status *appsv1.DeploymentStatus) {
	ts.Workloads[key.Name].DeploymentStatus = status
}

func (ts *testStatus) GetHealthStatus(key types.NamespacedName) string {
	return ts.Workloads[key.Name].HealthStatus
}

func (ts *testStatus) SetHealthStatus(key types.NamespacedName, status string) {
	ts.Workloads[key.Name].HealthStatus = status
}

func (ts *testStatus) GetHealthMessage(key types.NamespacedName) string {
	return ts.Workloads[key.Name].HealthMessage
}

func (ts *testStatus) SetHealthMessage(key types.NamespacedName, msg string) {
	ts.Workloads[key.Name].HealthMessage = msg
}

func (ts *testStatus) GetAggregatedHealthStatus() string {
	return ts.AggregatedHealth
}

func (ts *testStatus) SetAggregatedHealthStatus(status string) {
	ts.AggregatedHealth = status
}

func Test_setWorkloadHealth(t *testing.T) {

	type args struct {
		status AppStatusWithHealth
		obj    client.Object
	}
	tests := []struct {
		name       string
		args       args
		wantStatus *testStatus
		wantUpdate bool
		wantErr    bool
	}{
		{
			name: "Calculates workload health",
			args: args{
				status: &testStatus{
					AggregatedHealth: string(HealthStatusHealthy),
					Workloads: map[string]*testWorkloadStatus{
						"workload1": {},
					},
				},
				obj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workload1",
					},
					Status: appsv1.DeploymentStatus{
						ObservedGeneration:  4,
						Replicas:            2,
						UpdatedReplicas:     1,
						ReadyReplicas:       1,
						AvailableReplicas:   1,
						UnavailableReplicas: 1,
						Conditions: []appsv1.DeploymentCondition{
							{
								Type:    "Available",
								Status:  "True",
								Reason:  "MinimumReplicasAvailable",
								Message: "Deployment has minimum availability.",
							},
							{
								Type:    "Progressing",
								Status:  "True",
								Reason:  "ReplicaSetUpdated",
								Message: "ReplicaSet is progressing.",
							},
						},
					},
				},
			},
			wantStatus: &testStatus{
				AggregatedHealth: "Progressing",
				Workloads: map[string]*testWorkloadStatus{
					"workload1": {
						HealthStatus:  string(HealthStatusProgressing),
						HealthMessage: "Waiting for rollout to finish: 1 old replicas are pending termination...",
					},
				},
			},
			wantUpdate: true,
			wantErr:    false,
		},
		{
			name: "Worse workload health wins aggregation",
			args: args{
				status: &testStatus{
					AggregatedHealth: string(HealthStatusProgressing),
					Workloads: map[string]*testWorkloadStatus{
						"workload1": {},
					},
				},
				obj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workload1",
					},
					Status: appsv1.DeploymentStatus{
						ObservedGeneration:  4,
						Replicas:            2,
						UpdatedReplicas:     1,
						ReadyReplicas:       1,
						AvailableReplicas:   1,
						UnavailableReplicas: 1,
						Conditions: []appsv1.DeploymentCondition{
							{
								Type:    "Available",
								Status:  "True",
								Reason:  "MinimumReplicasAvailable",
								Message: "Deployment has minimum availability.",
							},
							{
								Type:    "Progressing",
								Status:  "False",
								Reason:  "ProgressDeadlineExceeded",
								Message: "ReplicaSet has timed out progressing.",
							},
						},
					},
				},
			},
			wantStatus: &testStatus{
				AggregatedHealth: string(HealthStatusDegraded),
				Workloads: map[string]*testWorkloadStatus{
					"workload1": {
						HealthStatus:  "Degraded",
						HealthMessage: `Deployment "workload1" exceeded its progress deadline`,
					},
				},
			},
			wantUpdate: true,
			wantErr:    false,
		},
		{
			name: "Better workload health looses aggregation",
			args: args{
				status: &testStatus{
					AggregatedHealth: string(HealthStatusUnknown),
					Workloads: map[string]*testWorkloadStatus{
						"workload1": {},
					},
				},
				obj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workload1",
					},
					Status: appsv1.DeploymentStatus{
						ObservedGeneration:  4,
						Replicas:            2,
						UpdatedReplicas:     1,
						ReadyReplicas:       1,
						AvailableReplicas:   1,
						UnavailableReplicas: 1,
						Conditions: []appsv1.DeploymentCondition{
							{
								Type:    "Available",
								Status:  "True",
								Reason:  "MinimumReplicasAvailable",
								Message: "Deployment has minimum availability.",
							},
							{
								Type:    "Progressing",
								Status:  "False",
								Reason:  "ProgressDeadlineExceeded",
								Message: "ReplicaSet has timed out progressing.",
							},
						},
					},
				},
			},
			wantStatus: &testStatus{
				AggregatedHealth: string(HealthStatusUnknown),
				Workloads: map[string]*testWorkloadStatus{
					"workload1": {
						HealthStatus:  string(HealthStatusDegraded),
						HealthMessage: `Deployment "workload1" exceeded its progress deadline`,
					},
				},
			},
			wantUpdate: true,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setWorkloadHealth(tt.args.status, tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("setWorkloadHealth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantUpdate {
				t.Errorf("setWorkloadHealth() = %v, want %v", got, tt.wantUpdate)
			}
			if diff := cmp.Diff(tt.args.status, tt.wantStatus); len(diff) > 0 {
				t.Errorf("setWorkloadHealth() = diff %s", diff)
			}
		})
	}
}
