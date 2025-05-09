/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"context"
	"crypto/rand"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/3scale-sre/basereconciler/reconciler"
	"github.com/goombaio/namegenerator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// +kubebuilder:scaffold:imports
	"github.com/3scale-sre/basereconciler/test/api/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	nameGenerator namegenerator.Generator
	timeout       time.Duration = 30 * time.Second
	poll          time.Duration = 5 * time.Second
	ctx           context.Context
	cancel        context.CancelFunc
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("api", "v1alpha1"),
		},
		// UseExistingCluster: ptr.To(true),
	}

	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	Expect(err).NotTo(HaveOccurred())
	nameGenerator = namegenerator.NewNameGenerator(nBig.Int64())

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		// Disable the metrics port to allow running the
		// test suite in parallel
		Metrics: metricsserver.Options{BindAddress: "0"},
	})

	Expect(err).ToNot(HaveOccurred())

	k8sClient = mgr.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	// Add controllers for testing
	err = (&Reconciler{
		Reconciler: reconciler.NewFromManager(mgr).WithLogger(ctrl.Log.WithName("controllers").WithName("Test")),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	Eventually(func() error {
		return testEnv.Stop()
	}, timeout, poll).ShouldNot(HaveOccurred())
})
