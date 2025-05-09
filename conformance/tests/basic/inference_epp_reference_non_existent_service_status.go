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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	inferenceapi "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"
	"sigs.k8s.io/gateway-api-inference-extension/conformance/tests"
	infrakubernetes "sigs.k8s.io/gateway-api-inference-extension/conformance/utils/kubernetes"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewaykubernetes "sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
)

func init() {
	tests.ConformanceTests = append(tests.ConformanceTests, InferencePoolEPPReferenceNonExistentServiceStatus)
}

// InferencePoolEPPReferenceNonExistentServiceStatus defines the test case for verifying
// InferencePool status when its extensionRef points to a non-existent EPP service.
var InferencePoolEPPReferenceNonExistentServiceStatus = suite.ConformanceTest{
	ShortName:   "InferencePoolEPPReferenceNonExistentServiceStatus",
	Description: "Validate InferencePool status reports an error when extensionRef points to a non-existent EPP service.",
	Manifests:   []string{"tests/basic/inference_epp_reference_non_existent_service_status.yaml"},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		poolNN := types.NamespacedName{
			Name:      "pool-non-existent-epp",
			Namespace: "gateway-conformance-app-backend",
		}
		routeNN := types.NamespacedName{
			Name:      "httproute-for-pool-non-existent-epp",
			Namespace: "gateway-conformance-app-backend",
		}
		gatewayNN := types.NamespacedName{
			Name:      "conformance-gateway",       // As defined in shared manifests
			Namespace: "gateway-conformance-infra", // As defined in shared manifests
		}

		httpRouteAcceptedCondition := metav1.Condition{
			Type:   string(gatewayv1.RouteConditionAccepted),
			Status: metav1.ConditionTrue,
			Reason: string(gatewayv1.RouteReasonAccepted),
		}

		// Step 1: Ensure the HTTPRoute is accepted by the Gateway. This makes the InferencePool
		// a recognized backend, which can be a prerequisite for the InferencePool
		// controller to fully process it and set its own status conditions.
		t.Logf("Waiting for HTTPRoute %s to be accepted by Gateway %s", routeNN.String(), gatewayNN.String())
		gatewaykubernetes.HTTPRouteMustHaveCondition(t, s.Client, s.TimeoutConfig, routeNN, gatewayNN, httpRouteAcceptedCondition)
		t.Logf("HTTPRoute %s is Accepted by Gateway %s", routeNN.String(), gatewayNN.String())

		// Step 2: Observe the status of the InferencePool "pool-non-existent-epp".
		// Expected: Its status should indicate a non-ready state because its extensionRef
		// points to a service that does not exist.
		// We expect a condition like: Type: "Accepted", Status: "False", Reason: "EPPServiceNotFound".

		expectedCondition := metav1.Condition{
			Type:   string(inferenceapi.InferencePoolConditionAccepted),
			Status: metav1.ConditionFalse,
			Reason: "EPPServiceNotFound", // Based on the expected reason from the test description.
		}

		t.Logf("Waiting for InferencePool %s to have condition: Type=%s, Status=%s, Reason=%s",
			poolNN.String(), expectedCondition.Type, expectedCondition.Status, expectedCondition.Reason)

		infrakubernetes.InferencePoolMustHaveCondition(t, s.Client, poolNN, expectedCondition)

		t.Logf("Successfully verified InferencePool %s has Type:%s Status:%s with Reason:%s",
			poolNN.String(), expectedCondition.Type, expectedCondition.Status, expectedCondition.Reason)
	},
}
