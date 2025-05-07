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

package config

import (
	"time"

	// Import the upstream Gateway API timeout config
	gatewayconfig "sigs.k8s.io/gateway-api/conformance/utils/config"
)

// InferenceExtensionTimeoutConfig embeds the upstream TimeoutConfig and adds
// extension-specific timeout values.
type InferenceExtensionTimeoutConfig struct {
	// All fields from gatewayconfig.TimeoutConfig will be available directly.
	gatewayconfig.TimeoutConfig

	// InferencePoolMustHaveConditionTimeout represents the maximum time to wait for an InferencePool to have a specific condition.
	InferencePoolMustHaveConditionTimeout time.Duration

	// InferencePoolMustHaveConditionInterval represents the polling interval for checking an InferencePool's condition.
	InferencePoolMustHaveConditionInterval time.Duration
}

func DefaultInferenceExtensionTimeoutConfig() InferenceExtensionTimeoutConfig {
	return InferenceExtensionTimeoutConfig{
		TimeoutConfig:                          gatewayconfig.DefaultTimeoutConfig(), // Get upstream defaults
		InferencePoolMustHaveConditionTimeout:  300 * time.Second,
		InferencePoolMustHaveConditionInterval: 10 * time.Second,
	}
}

// NewInferenceExtensionTimeoutConfigFromGatewayConfig creates a new InferenceExtensionTimeoutConfig
// using the provided gatewayconfig.TimeoutConfig for the embedded part and
// default values for the extension-specific fields.
func NewInferenceExtensionTimeoutConfigFromGatewayConfig(gc gatewayconfig.TimeoutConfig) InferenceExtensionTimeoutConfig {
	defaults := DefaultInferenceExtensionTimeoutConfig() // Your existing function for defaults
	return InferenceExtensionTimeoutConfig{
		TimeoutConfig:                          gc, // Use the (flag-aware) upstream config
		InferencePoolMustHaveConditionTimeout:  defaults.InferencePoolMustHaveConditionTimeout,
		InferencePoolMustHaveConditionInterval: defaults.InferencePoolMustHaveConditionInterval,
		// Initialize other extension-specific fields from 'defaults' as needed
	}
}
