// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build secrets

package secrets

import (
	"bytes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSecrets(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		out         string
		usePrefixes bool
		selector    backendSelector
		err         string
		skipWindows bool
	}{
		{
			name: "invalid input",
			in:   "invalid",
			out:  "",
			err:  "failed to unmarshal json input",
		},
		{
			name: "invalid version",
			in: `
			{
				"version": "2.0"
			}
			`,
			out: "",
			err: `incompatible protocol version "2.0"`,
		},
		{
			name: "no secrets",
			in: `
			{
				"version": "1.0"
			}
			`,
			out: "",
			err: `no secrets listed in input`,
		},
		{
			name: "valid input, reading from file",
			in: `
			{
				"version": "1.0",
				"secrets": [
					"secret1",
					"secret2",
					"secret3"
				]
			}
			`,
			out: `
			{
				"secret1": {
					"value": "secret1-value"
				},
				"secret2": {
					"error": "secret does not exist"
				},
				"secret3": {
					"error": "secret exceeds max allowed size"
				}
			}
			`,
		},
		{
			name:        "symlinks",
			skipWindows: true,
			in: `
			{
				"version": "1.0",
				"secrets": [
					"secret4",
					"secret5",
					"secret6"
				]
			}
			`,
			out: `
			{
				"secret4": {
					"value": "secret1-value"
				},
				"secret5": {
					"error": "not following symlink \"$TESTDATA/secret5-target\" outside of \"testdata/read-secrets\""
				},
				"secret6": {
					"error": "secret exceeds max allowed size"
				}
			}
			`,
		},
		{
			name: "valid input, reading from file and kube backends",
			in: `
			{
				"version": "1.0",
				"secrets": [
					"file/read-secrets/secret1",
					"kube_secret/some_namespace/some_name/some_key",
					"file/read-secrets/secret2",
					"kube_secret/another_namespace/another_name/another_key"
				]
			}
			`,
			out: `
			{
				"file/read-secrets/secret1": {
					"value": "secret1-value"
				},
				"kube_secret/some_namespace/some_name/some_key": {
					"value": "some_value"
				},
				"file/read-secrets/secret2": {
					"error": "secret does not exist"
				},
				"kube_secret/another_namespace/another_name/another_key": {
					"error": "secrets \"another_name\" not found"
				}
			}
			`,
			usePrefixes: true,
			selector: backendSelector{
				file: &fileBackend{rootPath: "./testdata"},
				kubeSecret: &kubeSecretBackend{
					kubeClient: fake.NewSimpleClientset(&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some_name",
							Namespace: "some_namespace",
						},
						Data: map[string][]byte{"some_key": []byte("some_value")},
					}),
				},
			},
		},
	}

	path := filepath.Join("testdata", "read-secrets")
	testdata, _ := filepath.Abs("testdata")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.skipWindows && runtime.GOOS == "windows" {
				t.Skip("skipped on windows")
			}
			var w bytes.Buffer
			err := decodeSecrets(strings.NewReader(test.in), &w, path, test.usePrefixes, &test.selector)
			out := string(w.Bytes())

			if test.out != "" {
				assert.JSONEq(t, strings.ReplaceAll(test.out, "$TESTDATA", testdata), out)
			} else {
				assert.Empty(t, out)
			}

			if test.err != "" {
				assert.EqualError(t, err, test.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
