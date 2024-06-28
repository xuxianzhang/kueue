/*
Copyright 2023 The Kubernetes Authors.

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

package provisioning

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	autoscaling "k8s.io/autoscaler/cluster-autoscaler/apis/provisioningrequest/autoscaling.x-k8s.io/v1beta1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/controller/admissionchecks/provisioning"
	"sigs.k8s.io/kueue/pkg/controller/core"
	"sigs.k8s.io/kueue/pkg/controller/core/indexer"
	"sigs.k8s.io/kueue/pkg/queue"
	"sigs.k8s.io/kueue/pkg/webhooks"
	"sigs.k8s.io/kueue/test/integration/framework"
	// +kubebuilder:scaffold:imports
)

var (
	cfg         *rest.Config
	k8sClient   client.Client
	ctx         context.Context
	fwk         *framework.Framework
	crdPath     = filepath.Join("..", "..", "..", "..", "..", "config", "components", "crd", "bases")
	depCRDPaths = []string{filepath.Join("..", "..", "..", "..", "..", "dep-crds", "cluster-autoscaler")}
	webhookPath = filepath.Join("..", "..", "..", "..", "..", "config", "components", "webhook")
)

func TestProvisioning(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t,
		"Provisioning admission check suite",
	)
}

func managerSetup(opts ...provisioning.Option) framework.ManagerSetup {
	return func(ctx context.Context, mgr manager.Manager) {
		err := indexer.Setup(ctx, mgr.GetFieldIndexer())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		failedWebhook, err := webhooks.Setup(mgr)
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "webhook", failedWebhook)

		controllersCfg := &config.Configuration{}
		mgr.GetScheme().Default(controllersCfg)

		controllersCfg.Metrics.EnableClusterQueueResources = true

		cCache := cache.New(mgr.GetClient())
		queues := queue.NewManager(mgr.GetClient(), cCache)

		failedCtrl, err := core.SetupControllers(mgr, queues, cCache, controllersCfg)
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "controller", failedCtrl)

		err = autoscaling.AddToScheme(mgr.GetScheme())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = provisioning.SetupIndexer(ctx, mgr.GetFieldIndexer())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		reconciler, err := provisioning.NewController(
			mgr.GetClient(),
			mgr.GetEventRecorderFor("kueue-provisioning-request-controller"),
			opts...,
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = reconciler.SetupWithManager(mgr)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
}
