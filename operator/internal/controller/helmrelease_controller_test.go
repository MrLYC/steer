/*
Copyright 2026 MrLYC.

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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
	"github.com/MrLYC/steer/operator/pkg/helm"
)

var _ = Describe("HelmRelease Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		helmrelease := &steerv1alpha1.HelmRelease{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind HelmRelease")
			err := k8sClient.Get(ctx, typeNamespacedName, helmrelease)
			if err != nil && errors.IsNotFound(err) {
				resource := &steerv1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: steerv1alpha1.HelmReleaseSpec{
						Chart: steerv1alpha1.ChartSpec{
							Source: steerv1alpha1.ChartSourceRepository,
							Repository: &steerv1alpha1.RepositoryChartSpec{
								URL:  "https://example.invalid/charts",
								Name: "example",
							},
						},
						Deployment: steerv1alpha1.DeploymentSpec{
							Namespace:       "default",
							CreateNamespace: false,
							Timeout:         metav1.Duration{Duration: 0},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &steerv1alpha1.HelmRelease{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance HelmRelease")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HelmReleaseReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Helm: &helm.FakeClient{
					InstallOrUpgradeFunc: func(ctx context.Context, req helm.InstallOrUpgradeRequest) (helm.ReleaseInfo, error) {
						return helm.ReleaseInfo{Name: req.ReleaseName, Namespace: req.Namespace, Version: 1, Status: "deployed"}, nil
					},
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &steerv1alpha1.HelmRelease{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(steerv1alpha1.HelmReleasePhaseInstalled))
			Expect(updated.Status.DeployedAt).NotTo(BeNil())
			Expect(updated.Status.HelmRelease).NotTo(BeNil())
			Expect(updated.Status.HelmRelease.Name).To(Equal(resourceName))
		})
	})
})
