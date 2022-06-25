package reconcilers

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/helpers"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Securityconfig Reconciler", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		clusterName = "securityconfig"
		timeout     = time.Second * 10
		interval    = time.Second * 1
	)

	Context("When Reconciling the securityconfig reconciler with no securityconfig provided", func() {
		It("should not do anything ", func() {
			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: clusterName, UID: "dummyuid"},
				Spec:       opsterv1.ClusterSpec{General: opsterv1.GeneralConfig{}}}

			reconcilerContext := NewReconcilerContext(spec.Spec.NodePools)
			underTest := NewSecurityconfigReconciler(
				k8sClient,
				context.Background(),
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)
			result, err := underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

		})
	})

	Context("When Reconciling the securityconfig reconciler with securityconfig secret configured but not available", func() {
		It("should trigger a requeue", func() {
			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: clusterName, UID: "dummyuid"},
				Spec: opsterv1.ClusterSpec{
					General: opsterv1.GeneralConfig{},
					Security: &opsterv1.Security{
						Config: &opsterv1.SecurityConfig{
							SecurityconfigSecret: corev1.LocalObjectReference{Name: "foobar"},
							AdminSecret:          corev1.LocalObjectReference{Name: "admin"},
						},
					},
				}}

			reconcilerContext := NewReconcilerContext(spec.Spec.NodePools)
			underTest := NewSecurityconfigReconciler(
				k8sClient,
				context.Background(),
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)
			result, err := underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsZero()).To(BeFalse())
			Expect(result.Requeue).To(BeTrue())
		})
	})

	Context("When Reconciling the securityconfig reconciler with admin secret configured and available", func() {
		It("should start an update job", func() {
			var clusterName = "securityconfig-withadminsecret"
			// Create namespace and secrets first
			Expect(CreateNamespace(k8sClient, clusterName)).Should(Succeed())

			adminCertSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "admin-cert", Namespace: clusterName},
				Type:       corev1.SecretType("Opaque"),
				Data:       map[string][]byte{},
			}
			err := k8sClient.Create(context.Background(), adminCertSecret)
			Expect(err).ToNot(HaveOccurred())

			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: clusterName, UID: "dummyuid"},
				Spec: opsterv1.ClusterSpec{
					General: opsterv1.GeneralConfig{},
					Security: &opsterv1.Security{
						Config: &opsterv1.SecurityConfig{
							SecurityconfigSecret: corev1.LocalObjectReference{Name: "foobar"},
							AdminSecret:          corev1.LocalObjectReference{Name: "admin-cert"},
						},
					},
				}}

			reconcilerContext := NewReconcilerContext(spec.Spec.NodePools)
			underTest := NewSecurityconfigReconciler(
				k8sClient,
				context.Background(),
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)
			_, err = underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())

			job := batchv1.Job{}
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterName + "-securityconfig-update", Namespace: clusterName}, &job)).To(HaveOccurred())

		})
	})

	Context("When Reconciling the securityconfig reconciler with securityconfig secret but no adminSecret configured", func() {
		It("should not start an update job", func() {
			var clusterName = "securityconfig-noadminsecret"
			// Create namespace and secret first
			Expect(CreateNamespace(k8sClient, clusterName)).Should(Succeed())
			configSecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "securityconfig", Namespace: clusterName},
				StringData: map[string]string{"config.yml": "foobar"},
			}
			err := k8sClient.Create(context.Background(), &configSecret)
			Expect(err).ToNot(HaveOccurred())

			spec := opsterv1.OpenSearchCluster{
				ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: clusterName, UID: "dummyuid"},
				Spec: opsterv1.ClusterSpec{
					General: opsterv1.GeneralConfig{},
					Security: &opsterv1.Security{
						Config: &opsterv1.SecurityConfig{
							SecurityconfigSecret: corev1.LocalObjectReference{Name: "securityconfig"},
						},
					},
				}}

			reconcilerContext := NewReconcilerContext(spec.Spec.NodePools)
			underTest := NewSecurityconfigReconciler(
				k8sClient,
				context.Background(),
				&helpers.MockEventRecorder{},
				&reconcilerContext,
				&spec,
			)
			_, err = underTest.Reconcile()
			Expect(err).ToNot(HaveOccurred())

			job := batchv1.Job{}
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterName + "-securityconfig-update", Namespace: clusterName}, &job)).To(HaveOccurred())

		})
	})
})
