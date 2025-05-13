/*
Copyright 2025 The Kubernetes Authors.

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

package basic

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"

	// Import the tests package to append to ConformanceTests
	"sigs.k8s.io/gateway-api-inference-extension/conformance/tests"
	infrakubernetes "sigs.k8s.io/gateway-api-inference-extension/conformance/utils/kubernetes"
	gatewaykubernetes "sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
)

func init() {
	tests.ConformanceTests = append(tests.ConformanceTests, InferencePoolResolvedRefsCondition)
}

// InferencePoolResolvedRefsCondition defines the test case for verifying
// that an InferencePool correctly surfaces the "ResolvedRefs" condition type
// as it is referenced by other Gateway API resources.
var InferencePoolResolvedRefsCondition = suite.ConformanceTest{
	ShortName:   "InferencePoolResolvedRefsCondition",
	Description: "Verify that an InferencePool correctly surfaces the 'ResolvedRefs' condition type, indicating whether it is successfully referenced by other Gateway API resources.",
	Manifests:   []string{"/conformance/tests/basic/inferencepool_resolvedrefs_condition.yaml"},
	Features:    []features.FeatureName{},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		const (
			appBackendNamespace = "gateway-conformance-app-backend"
			infraNamespace      = "gateway-conformance-infra"
			poolName            = "multi-gateway-pool"
			gateway1Name        = "conformance-gateway" // From base manifests
			gateway2Name        = "gateway-2"           // Defined in test manifest
			httpRoute1Name      = "httproute-for-gw1"
			httpRoute2Name      = "httproute-for-gw2"

			// Expected Reasons for ResolvedRefs condition
			reasonRefsResolved = "RefsResolved"
			reasonNoRefsFound  = "NoRefsFound"
		)

		poolNN := types.NamespacedName{Name: poolName, Namespace: appBackendNamespace}
		httpRoute1NN := types.NamespacedName{Name: httpRoute1Name, Namespace: appBackendNamespace}
		httpRoute2NN := types.NamespacedName{Name: httpRoute2Name, Namespace: appBackendNamespace}

		// Define the expected "Accepted" condition for HTTPRoutes
		acceptedCondition := metav1.Condition{
			Type:   string(gatewayv1.RouteConditionAccepted),
			Status: metav1.ConditionTrue,
			Reason: string(gatewayv1.RouteReasonAccepted),
		}

		t.Logf("Waiting for HTTPRoute %s to be Accepted by Gateway %s", httpRoute1NN.String(), gateway1Name)
		gatewaykubernetes.HTTPRouteMustHaveCondition(t, s.Client, s.TimeoutConfig, httpRoute1NN, types.NamespacedName{Name: gateway1Name, Namespace: infraNamespace}, acceptedCondition)
		t.Logf("Waiting for HTTPRoute %s to be Accepted by Gateway %s", httpRoute2NN.String(), gateway2Name)
		gatewaykubernetes.HTTPRouteMustHaveCondition(t, s.Client, s.TimeoutConfig, httpRoute2NN, types.NamespacedName{Name: gateway2Name, Namespace: appBackendNamespace}, acceptedCondition)

		// Step 3: Observe "multi-gateway-pool" status (Initial state - 2 HTTPRoutes referencing it).
		// Expected: ResolvedRefs: True, Reason: RefsResolved, and Message indicating multiple references.
		t.Run("InferencePool should show ResolvedRefs: True when referenced by multiple HTTPRoutes", func(t *testing.T) {
			expectedCondition := metav1.Condition{
				Type:   string(gatewayv1.RouteConditionResolvedRefs), // Use standard Gateway API condition type for references
				Status: metav1.ConditionTrue,
				Reason: reasonRefsResolved,
				// Open action item: API to define exact multi-reference messaging.
				// For now, we just assert presence of True and Reason.
				// If a specific message substring is expected, add it here.
			}
			infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, expectedCondition)
			t.Logf("InferencePool %s has ResolvedRefs: True as expected with two references.", poolNN.String())
		})

		// Step 4: Delete "httproute-for-gw1".
		t.Run("Delete httproute-for-gw1", func(t *testing.T) {
			httproute1 := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      httpRoute1NN.Name,
					Namespace: httpRoute1NN.Namespace,
				},
			}
			t.Logf("Deleting HTTPRoute %s", httpRoute1NN.String())
			require.NoError(t, s.Client.Delete(context.TODO(), httproute1), "failed to delete httproute-for-gw1")
			// Give the controller some time to process the deletion
			time.Sleep(s.TimeoutConfig.GatewayMustHaveCondition)
		})

		// Step 5: Observe "multi-gateway-pool" status (After deleting 1st HTTPRoute).
		// Expected: ResolvedRefs: True, Message updated to reflect remaining reference count.
		t.Run("InferencePool should still show ResolvedRefs: True after one HTTPRoute is deleted", func(t *testing.T) {
			expectedCondition := metav1.Condition{
				Type:   string(gatewayv1.RouteConditionResolvedRefs),
				Status: metav1.ConditionTrue,
				Reason: reasonRefsResolved,
			}
			infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, expectedCondition)
			t.Logf("InferencePool %s still has ResolvedRefs: True as expected with one reference remaining.", poolNN.String())
		})

		// Step 6: Delete "httproute-for-gw2".
		t.Run("Delete httproute-for-gw2", func(t *testing.T) {
			httproute2 := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      httpRoute2NN.Name,
					Namespace: httpRoute2NN.Namespace,
				},
			}
			t.Logf("Deleting HTTPRoute %s", httpRoute2NN.String())
			require.NoError(t, s.Client.Delete(context.TODO(), httproute2), "failed to delete httproute-for-gw2")
			// Give the controller some time to process the deletion
			time.Sleep(s.TimeoutConfig.GatewayMustHaveCondition)
		})

		// Step 7: Observe "multi-gateway-pool" status (After deleting 2nd HTTPRoute).
		// Expected: ResolvedRefs: False, Reason: NoRefsFound, and Message indicating no references.
		t.Run("InferencePool should show ResolvedRefs: False after all HTTPRoutes are deleted", func(t *testing.T) {
			expectedCondition := metav1.Condition{
				Type:   string(gatewayv1.RouteConditionResolvedRefs),
				Status: metav1.ConditionFalse,
				Reason: reasonNoRefsFound,
			}
			infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, expectedCondition)
			t.Logf("InferencePool %s has ResolvedRefs: False as expected with no references.", poolNN.String())
		})

		t.Logf("TestInferencePoolResolvedRefsCondition completed.")
	},
}
