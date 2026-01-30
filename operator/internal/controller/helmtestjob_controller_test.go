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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
)

var _ = Describe("HelmTestJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		helmtestjob := &steerv1alpha1.HelmTestJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind HelmTestJob")
			err := k8sClient.Get(ctx, typeNamespacedName, helmtestjob)
			if err != nil && errors.IsNotFound(err) {
				resource := &steerv1alpha1.HelmTestJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: steerv1alpha1.HelmTestJobSpec{
						HelmReleaseRef: steerv1alpha1.HelmReleaseRef{
							Name:      "example-release",
							Namespace: "default",
						},
						Schedule: steerv1alpha1.ScheduleSpec{
							Type: steerv1alpha1.ScheduleTypeOnce,
							Delay: metav1.Duration{
								Duration: 0,
							},
							Timezone: "Asia/Shanghai",
						},
						Test: steerv1alpha1.TestSpec{
							Timeout: metav1.Duration{Duration: 0},
							Logs:    nil,
							Filter:  "",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &steerv1alpha1.HelmTestJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance HelmTestJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the once schedule resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HelmTestJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &steerv1alpha1.HelmTestJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(steerv1alpha1.HelmTestJobPhasePending))
			Expect(updated.Status.NextScheduleTime).NotTo(BeNil())
		})

		It("should successfully reconcile the cron schedule resource", func() {
			By("Updating the resource to use cron schedule")
			resource := &steerv1alpha1.HelmTestJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			resource.Spec.Schedule.Type = steerv1alpha1.ScheduleTypeCron
			resource.Spec.Schedule.Cron = "* * * * *"
			resource.Spec.Schedule.Timezone = "Asia/Shanghai"
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			controllerReconciler := &HelmTestJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &steerv1alpha1.HelmTestJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.NextScheduleTime).NotTo(BeNil())
			// Cron should schedule the next minute boundary (or later) in the specified timezone.
			// Allow small clock skew to avoid flakiness.
			Expect(updated.Status.NextScheduleTime.Time.After(time.Now().Add(-5 * time.Second))).To(BeTrue())
		})
	})
})
