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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"

	// inferenceapi "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2" // If you need to Get the InferencePool typed object
	"sigs.k8s.io/gateway-api-inference-extension/conformance/tests"
	infrakubernetes "sigs.k8s.io/gateway-api-inference-extension/conformance/utils/kubernetes"
)

func init() {
	tests.ConformanceTests = append(tests.ConformanceTests, InferencePoolNoMatchingPodsRouteStatus)
}

var InferencePoolNoMatchingPodsRouteStatus = suite.ConformanceTest{
	ShortName:   "InferencePoolNoMatchingPodsRouteStatus",
	Description: "Tests HTTPRoute and Gateway status when an HTTPRoute references an InferencePool whose modelServerSelector does not match any running pods.",
	Manifests:   []string{"tests/basic/inferencepool_no_matching_pods_route_status.yaml"}, // References the new YAML
	Features:    []features.FeatureName{},                                                 // Add specific features if this test targets them
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		const (
			appBackendNamespace   = "gateway-conformance-app-backend"
			infraNamespace        = "gateway-conformance-infra"
			poolName              = "pool-no-pods"
			httpRouteName         = "httproute-for-pool-no-pods"
			gatewayName           = "conformance-gateway"
			expectedRouteReason   = "ReconciliationFailed" // Based on previous logs for GKE Gateway
			expectedGatewayReason = "Invalid"              // Based on previous logs for GKE Gateway
			pollingInterval       = 5 * time.Second
			timeout               = 2 * time.Minute // Adjust if necessary
		)

		poolNN := types.NamespacedName{Name: poolName, Namespace: appBackendNamespace}
		routeNN := types.NamespacedName{Name: httpRouteName, Namespace: appBackendNamespace}
		gatewayNN := types.NamespacedName{Name: gatewayName, Namespace: infraNamespace}

		t.Logf("Manifests applied. Waiting for controllers to process InferencePool %s and HTTPRoute %s", poolNN.String(), routeNN.String())

		// Step 1: Verify initial acceptance of the InferencePool by the Gateway (via the HTTPRoute)
		// The InferencePool's .status.parent field should show it's Accepted by the Gateway.
		// This doesn't mean the InferencePool is ready, just that the Gateway acknowledges the HTTPRoute using it.
		t.Logf("Verifying InferencePool %s is initially accepted by Gateway %s (via HTTPRoute %s)", poolNN.String(), gatewayNN.String(), routeNN.String())
		acceptedCondition := metav1.Condition{
			Type:   string(gatewayv1.RouteConditionAccepted), // For the route parent status on InferencePool
			Status: metav1.ConditionTrue,
			Reason: string(gatewayv1.RouteReasonAccepted),
		}
		// Note: The infrakubernetes.InferencePoolMustHaveCondition checks .status.parent.conditions
		// This helper might need adjustment if it looks for top-level conditions on InferencePool.
		// For now, we assume it correctly checks the parent-relative status as seen in user's YAML.
		infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, acceptedCondition)
		t.Logf("InferencePool %s parent status shows Accepted by Gateway %s", poolNN.String(), gatewayNN.String())

		// Step 2: Observe the status of the HTTPRoute
		// Expect the HTTPRoute to be Accepted but fail Reconciliation due to backend issues.
		t.Logf("Polling for HTTPRoute %s status to reflect backend issues...", routeNN.String())

		var lastObservedRouteStatus string
		require.Eventually(t, func() bool {
			route := &gatewayv1.HTTPRoute{}
			err := s.Client.Get(context.TODO(), routeNN, route)
			if err != nil {
				t.Logf("Error getting HTTPRoute %s: %v. Retrying...", routeNN.String(), err)
				return false
			}

			// Log current status for debugging if it changes
			currentStatusStr := fmt.Sprintf("%+v", route.Status.Parents)
			if currentStatusStr != lastObservedRouteStatus {
				t.Logf("Current HTTPRoute %s parent statuses: %s", routeNN.String(), currentStatusStr)
				lastObservedRouteStatus = currentStatusStr
			}

			var relevantParentStatus *gatewayv1.RouteParentStatus
			for _, parentStatus := range route.Status.Parents {
				if parentStatus.ParentRef.Name == gatewayv1.ObjectName(gatewayName) &&
					parentStatus.ParentRef.Namespace != nil && *parentStatus.ParentRef.Namespace == gatewayv1.Namespace(infraNamespace) &&
					string(parentStatus.ControllerName) == s.ControllerName { // Check against the suite's controller name
					relevantParentStatus = &parentStatus
					break
				}
			}

			if relevantParentStatus == nil {
				t.Logf("Relevant parent status for Gateway %s not yet found in HTTPRoute %s. Retrying...", gatewayNN.String(), routeNN.String())
				return false
			}

			// Check for Accepted: True
			accepted := kubernetes.FindCondition(relevantParentStatus.Conditions, gatewayv1.RouteConditionAccepted)
			if accepted == nil || accepted.Status != metav1.ConditionTrue {
				t.Logf("HTTPRoute %s not yet Accepted or Accepted status is not True. Accepted Condition: %+v. Retrying...", routeNN.String(), accepted)
				return false
			}

			// Check for Reconciled: False with the specific reason
			reconciled := kubernetes.FindCondition(relevantParentStatus.Conditions, gatewayv1.RouteConditionResolvedRefs) // Using ResolvedRefs as a proxy if Reconciled is not standard or consistently set for failure by all controllers.
			// Or, more directly, gatewayv1.RouteConditionProgrammed if available and used for failure.
			// For GKE, "Reconciled" was the condition type seen in logs for the route's status relative to the Gateway.
			// Let's stick to what was observed: the HTTPRoute's parent status `Reconciled` condition.
			reconciledOnRoute := kubernetes.FindCondition(relevantParentStatus.Conditions, gatewayv1.RouteConditionType("Reconciled"))

			if reconciledOnRoute != nil {
				t.Logf("HTTPRoute %s Reconciled Condition: Status=%s, Reason=%s, Message=%s",
					routeNN.String(), reconciledOnRoute.Status, reconciledOnRoute.Reason, reconciledOnRoute.Message)
				if reconciledOnRoute.Status == metav1.ConditionFalse && string(reconciledOnRoute.Reason) == expectedRouteReason {
					// Check message for more specific details if needed
					// Based on previous logs: "error cause: no-error-isolation: missing neg status in annotation of extension service gateway-conformance-app-backend/pool-no-pods-epp"
					expectedMsgPart1 := "missing neg status in annotation of extension service"
					expectedMsgPart2 := fmt.Sprintf("%s/%s-epp", appBackendNamespace, poolName)

					if strings.Contains(reconciledOnRoute.Message, expectedMsgPart1) &&
						strings.Contains(reconciledOnRoute.Message, expectedMsgPart2) {
						t.Logf("SUCCESS: HTTPRoute %s has Reconciled:False with Reason:%s and expected error message parts.", routeNN.String(), expectedRouteReason)
						return true // Condition met
					}
					t.Logf("HTTPRoute %s Reconciled:False, Reason:%s, but message mismatch. Message: '%s'. Retrying...", routeNN.String(), expectedRouteReason, reconciledOnRoute.Message)
				} else {
					t.Logf("HTTPRoute %s Reconciled status is %s (expected False) or Reason is %s (expected %s). Retrying...", routeNN.String(), reconciledOnRoute.Status, reconciledOnRoute.Reason, expectedRouteReason)
				}
			} else {
				t.Logf("HTTPRoute %s Reconciled condition not yet found for Gateway %s. Retrying...", routeNN.String(), gatewayNN.String())
			}
			return false
		}, timeout, pollingInterval, "timed out waiting for HTTPRoute status to reflect backend issues")

		// Step 3: (Optional but Recommended) Observe the status of the parent Gateway
		t.Logf("Polling for Gateway %s/%s status to reflect issues...", gatewayNN.Namespace, gatewayNN.Name)
		var lastObservedGatewayStatus string
		require.Eventually(t, func() bool {
			gateway := &gatewayv1.Gateway{}
			err := s.Client.Get(context.TODO(), gatewayNN, gateway)
			if err != nil {
				t.Logf("Error getting Gateway %s: %v. Retrying...", gatewayNN.String(), err)
				return false
			}

			currentStatusStr := fmt.Sprintf("%+v", gateway.Status.Conditions)
			if currentStatusStr != lastObservedGatewayStatus {
				t.Logf("Current Gateway %s conditions: %s", gatewayNN.String(), currentStatusStr)
				lastObservedGatewayStatus = currentStatusStr
			}

			programmed := kubernetes.FindCondition(gateway.Status.Conditions, gatewayv1.GatewayConditionProgrammed)
			if programmed != nil {
				t.Logf("Gateway %s Programmed Condition: Status=%s, Reason=%s, Message=%s",
					gatewayNN.String(), programmed.Status, programmed.Reason, programmed.Message)
				if programmed.Status == metav1.ConditionFalse && string(programmed.Reason) == expectedGatewayReason {
					// Check message for specifics related to the problematic HTTPRoute or its -epp service
					// Based on previous logs: "...translateInferencePoolLbTrafficExtension for gateway-conformance-app-backend/pool-no-pods: port 9002 does not exist in service..."
					expectedErrPart1 := fmt.Sprintf("%s/%s-epp", appBackendNamespace, poolName) // e.g., pool-no-pods-epp
					expectedErrPart2 := "port 9002 does not exist"

					if strings.Contains(programmed.Message, expectedErrPart1) ||
						strings.Contains(programmed.Message, expectedErrPart2) ||
						strings.Contains(programmed.Message, httpRouteName) { // Mention of the problematic route
						t.Logf("SUCCESS: Gateway %s has Programmed:False with Reason:%s and relevant error message.", gatewayNN.String(), expectedGatewayReason)
						return true // Condition met
					}
					t.Logf("Gateway %s Programmed:False, Reason:%s, but message details mismatch. Message: '%s'. Retrying...", gatewayNN.String(), expectedGatewayReason, programmed.Message)
				} else {
					t.Logf("Gateway %s Programmed:False, but Reason is '%s' (expected %s) or Status is %s (expected False). Retrying...", gatewayNN.String(), programmed.Reason, expectedGatewayReason, programmed.Status)
				}
			} else {
				t.Logf("Gateway %s Programmed condition not yet found. Retrying...", gatewayNN.String())
			}
			return false
		}, timeout, pollingInterval, "timed out waiting for Gateway Programmed:False status")

		t.Logf("TestInferencePoolNoMatchingPodsRouteStatus completed.")
	},
}
