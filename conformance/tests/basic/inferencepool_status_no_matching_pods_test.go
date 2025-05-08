package basic

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	// Adjust the import path if your API types are located elsewhere.
	inferenceapi "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"
	"sigs.k8s.io/gateway-api-inference-extension/conformance/tests"
	infrakubernetes "sigs.k8s.io/gateway-api-inference-extension/conformance/utils/kubernetes"
	gatewaykubernetes "sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
)

func init() {
	tests.ConformanceTests = append(tests.ConformanceTests, InferencePoolStatusNoMatchingPods)
}

var InferencePoolStatusNoMatchingPods = suite.ConformanceTest{
	ShortName:   "InferencePoolStatusNoMatchingPods",
	Description: "Validate InferencePool status when modelServerSelector does not match any running pods.",
	Manifests:   []string{"../../tests/basic/inferencepool_status_no_matching_pods.yaml"},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		t.Run("InferencePool pool-no-pods status indicates no matching pods", func(t *testing.T) {
			poolNN := types.NamespacedName{
				Name:      "pool-no-pods",
				Namespace: "gateway-conformance-app-backend",
			}
			routeNN := types.NamespacedName{
				Name:      "httproute-for-pool-no-pods",
				Namespace: "gateway-conformance-app-backend",
			}
			gatewayNN := types.NamespacedName{
				Name:      "conformance-gateway",       // As defined in shared manifests
				Namespace: "gateway-conformance-infra", // As defined in shared manifests
			}

			// Step 1: Create an InferencePool "pool-no-pods" (done by manifest loading).
			// Expected: "pool-no-pods" creation is successful.

			// Ensure the HTTPRoute is accepted by the Gateway. This makes the InferencePool
			// a recognized backend, which can be a prerequisite for the InferencePool
			// controller to fully process it and set its own status conditions.
			t.Logf("Waiting for HTTPRoute %s to be accepted by Gateway %s", routeNN.String(), gatewayNN.String())
			gatewaykubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, s.Client, s.TimeoutConfig, routeNN, gatewayNN)

			// Ensure the InferencePool itself is marked as "Accepted" by its controller.
			// This means the InferencePool controller has acknowledged and validated its spec.
			// We expect a condition like: Type: "Accepted", Status: "True", Reason: "Accepted".
			// Replace "Accepted" (Type) and "Accepted" (Reason) with inferenceapi constants if available.
			acceptedCondition := metav1.Condition{
				Type:   string(inferenceapi.InferencePoolConditionAccepted), // e.g., "Accepted" or inferenceapi.InferencePoolConditionAccepted
				Status: metav1.ConditionTrue,
				Reason: string(inferenceapi.InferencePoolReasonAccepted), // e.g., "Accepted" or inferenceapi.InferencePoolReasonAccepted
			}
			t.Logf("Waiting for InferencePool %s to have condition: Type=%s, Status=%s, Reason=%s",
				poolNN.String(), acceptedCondition.Type, acceptedCondition.Status, acceptedCondition.Reason)
			// Pass s.Client and s.TimeoutConfig as Eventually helpers typically need them.
			infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, acceptedCondition)
			t.Logf("InferencePool %s is Accepted", poolNN.String())

			// Step 2: Observe the status of "pool-no-pods".
			// Expected: Its status should indicate a non-ready state because no pods match its selector.
			// We expect a condition like: Type: "Ready", Status: "False", Reason: "NoMatchingPods".

			notReadyCondition := metav1.Condition{
				Type:   string(inferenceapi.InferencePoolConditionAccepted),
				Status: metav1.ConditionFalse,
				Reason: "NoMatchingPods", // As per design doc; confirm with API spec
			}

			t.Logf("Waiting for InferencePool %s to have condition: Type=%s, Status=%s, Reason=%s",
				poolNN.String(), notReadyCondition.Type, notReadyCondition.Status, notReadyCondition.Reason)

			infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, notReadyCondition)

			t.Logf("Successfully verified InferencePool %s has Ready:False with Reason:%s", poolNN.String(), notReadyCondition.Reason)
		})
	},
}
