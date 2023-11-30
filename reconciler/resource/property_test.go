package resource

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
)

func TestProperty_Reconcile(t *testing.T) {
	type args struct {
		u_live           map[string]any
		u_desired        map[string]any
		u_normalizedLive map[string]any
		logger           logr.Logger
	}
	tests := []struct {
		name               string
		p                  Property
		args               args
		wantErr            bool
		wantLive           map[string]any
		wantNormalizedLive map[string]any
	}{
		{
			name: "PresentInBoth",
			p:    "a.b.c",
			args: args{
				u_live:           map[string]any{"a": map[string]any{"b": map[string]any{"c": "value", "d": 1}}},
				u_desired:        map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue"}}},
				u_normalizedLive: map[string]any{},
				logger:           logr.Discard(),
			},
			wantErr:            false,
			wantLive:           map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue", "d": 1}}},
			wantNormalizedLive: map[string]any{"a": map[string]any{"b": map[string]any{"c": "value"}}},
		},
		{
			name: "MissingInBoth",
			p:    "a.b.c",
			args: args{
				u_live:           map[string]any{"a": map[string]any{"b": map[string]any{"d": 1}}},
				u_desired:        map[string]any{},
				u_normalizedLive: map[string]any{},
				logger:           logr.Discard(),
			},
			wantErr:            false,
			wantLive:           map[string]any{"a": map[string]any{"b": map[string]any{"d": 1}}},
			wantNormalizedLive: map[string]any{},
		},
		{
			name: "PresentInDesiredMissingFromLive",
			p:    "a.b.c",
			args: args{
				u_live:           map[string]any{"a": map[string]any{}},
				u_desired:        map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue"}}},
				u_normalizedLive: map[string]any{},
				logger:           logr.Discard(),
			},
			wantErr:            false,
			wantLive:           map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue"}}},
			wantNormalizedLive: map[string]any{},
		},
		{
			name: "MissingFromDesiredPresentInLive",
			p:    "a.b.c",
			args: args{
				u_live:           map[string]any{"a": map[string]any{"b": map[string]any{"c": "value", "d": 1}}},
				u_desired:        map[string]any{"a": map[string]any{}},
				u_normalizedLive: map[string]any{},
				logger:           logr.Discard(),
			},
			wantErr:            false,
			wantLive:           map[string]any{"a": map[string]any{"b": map[string]any{"d": 1}}},
			wantNormalizedLive: map[string]any{"a": map[string]any{"b": map[string]any{"c": "value"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Reconcile(tt.args.u_live, tt.args.u_desired, tt.args.u_normalizedLive, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("Property.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.args.u_live, tt.wantLive); len(diff) > 0 {
				t.Errorf("Property.Reconcile() diff in live %v", diff)
			}
			if diff := cmp.Diff(tt.args.u_normalizedLive, tt.wantNormalizedLive); len(diff) > 0 {
				t.Errorf("Property.Reconcile() diff in normalizedLive  %v", diff)
			}
		})
	}
}

func Test_delta(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(delta(0, 0)).To(gomega.Equal(MissingInBoth))
	g.Expect(delta(0, 1)).To(gomega.Equal(MissingFromDesiredPresentInLive))
	g.Expect(delta(1, 0)).To(gomega.Equal(PresentInDesiredMissingFromLive))
	g.Expect(delta(1, 1)).To(gomega.Equal(PresentInBoth))
}
