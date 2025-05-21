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

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"

	// Import the tests package to append to ConformanceTests
	inferenceapi "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"
	"sigs.k8s.io/gateway-api-inference-extension/conformance/tests"
	k8utils "sigs.k8s.io/gateway-api-inference-extension/conformance/utils/kubernetes"
)

func init() {
	tests.ConformanceTests = append(tests.ConformanceTests, InferenceModelAccepted)
}

var InferenceModelAccepted = suite.ConformanceTest{
	ShortName:   "InferenceModelAccepted",
	Description: "Basic Create and Read operations for InferenceModel.",
	Manifests:   []string{"tests/basic/inferencesmodel_accepted.yaml"},
	Features: []features.FeatureName{
		features.FeatureName("SupportInferenceModel"),
		features.FeatureName("SupportInferencePool"),
	},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		const (
			modelNamespace    = "gateway-conformance-app-backend"
			modelMetaName     = "my-chat-model"
			modelSpecName     = "chat-model-v1"
			inferencePoolName = "test-pool-for-model"
		)

		modelNN := types.NamespacedName{Name: modelMetaName, Namespace: modelNamespace}

		t.Run("Step 1: Read created InferenceModel and verify spec", func(t *testing.T) {
			createdModel := &inferenceapi.InferenceModel{}
			err := s.Client.Get(context.TODO(), modelNN, createdModel)
			require.NoError(t, err, "failed to get InferenceModel resource")

			require.Equal(t, modelSpecName, createdModel.Spec.ModelName, "Read InferenceModel.Spec.ModelName does not match expected")

			expectedPoolRef := inferenceapi.PoolObjectReference{
				Group: "inference.networking.x-k8s.io",
				Kind:  "InferencePool",
				Name:  inferencePoolName,
			}
			require.Equal(t, string(expectedPoolRef.Group), string(createdModel.Spec.PoolRef.Group), "Read InferenceModel.Spec.PoolRef.Group does not match expected")
			require.Equal(t, string(expectedPoolRef.Kind), string(createdModel.Spec.PoolRef.Kind), "Read InferenceModel.Spec.PoolRef.Kind does not match expected")
			require.Equal(t, expectedPoolRef.Name, createdModel.Spec.PoolRef.Name, "Read InferenceModel.Spec.PoolRef.Name does not match expected")

			t.Logf("Successfully read and verified InferenceModel %s spec.", modelNN.String())
		})

		t.Run("Step 2: InferenceModel should have Accepted condition set to True", func(t *testing.T) {
			acceptedCondition := metav1.Condition{
				Type:   string(inferenceapi.ModelConditionAccepted),
				Status: metav1.ConditionTrue,
				Reason: string(inferenceapi.ModelReasonAccepted),
			}
			k8utils.InferenceModelMustHaveCondition(t, s.Client, modelNN, acceptedCondition)
			t.Logf("Verified InferenceModel %s has Condition %s: True with Reason: %s.",
				modelNN.String(), inferenceapi.ModelConditionAccepted, inferenceapi.ModelReasonAccepted)
		})
	},
}
