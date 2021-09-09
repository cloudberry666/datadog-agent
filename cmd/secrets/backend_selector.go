// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build secrets

package secrets

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/apiserver"
	"time"
)

type backendPrefix string

const (
	file       backendPrefix = "file"
	kubeSecret backendPrefix = "kube_secret"
)

func (prefix backendPrefix) isValid() bool {
	return prefix == file || prefix == kubeSecret
}

type backendSelector struct {
	file       *fileBackend
	kubeSecret *kubeSecretBackend
}

func (selector *backendSelector) choose(prefix backendPrefix) (backend, error) {
	// Lazy instantiate backends. We don't need to wait for the creation of a
	// kubernetes client until we need it, for example.
	switch prefix {
	case file:
		if selector.file == nil {
			// Assumes that / is always the root path if using the file backend.
			selector.file = &fileBackend{rootPath: "/"}
		}

		return selector.file, nil
	case kubeSecret:
		if selector.kubeSecret == nil {
			kubeClient, err := apiserver.GetKubeClient(10 * time.Second)
			if err != nil {
				return nil, err
			}

			selector.kubeSecret = &kubeSecretBackend{kubeClient: kubeClient}
		}

		return selector.kubeSecret, nil
	default:
		return nil, fmt.Errorf("backend not supported: %s", prefix)
	}
}
