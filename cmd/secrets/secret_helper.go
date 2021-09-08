// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build secrets

package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	s "github.com/DataDog/datadog-agent/pkg/secrets"
)

func init() {
	SecretHelperCmd.AddCommand(readSecretCmd)
}

// SecretHelperCmd implements secrets backend helper commands
var SecretHelperCmd = &cobra.Command{
	Use:   "secret-helper",
	Short: "Secret management backend helper",
	Long:  ``,
}

// ReadSecretsCmd implements reading secrets from a directory/volume mount
var readSecretCmd = &cobra.Command{
	Use:   "read",
	Short: "Read secret from a directory",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return readSecrets(os.Stdin, os.Stdout, args[0])
	},
}

type secretsRequest struct {
	Version string   `json:"version"`
	Secrets []string `json:"secrets"`
}

type backend interface {
	// secretID has a different format on each secret backend. In the file
	// backend, it's a path, whereas in the kubernetes secret one it has the
	// following format: "namespace/name/key".
	get(secretID string) s.Secret
}

func readSecrets(r io.Reader, w io.Writer, dir string) error {
	in, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	var request secretsRequest
	err = json.Unmarshal(in, &request)
	if err != nil {
		return errors.New("failed to unmarshal json input")
	}

	version := splitVersion(request.Version)
	compatVersion := splitVersion(s.PayloadVersion)
	if version[0] != compatVersion[0] {
		return fmt.Errorf("incompatible protocol version %q", request.Version)
	}

	if len(request.Secrets) == 0 {
		return errors.New("no secrets listed in input")
	}

	response := fetchSecretsUsingFile(request.Secrets, dir)

	out, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

func fetchSecretsUsingFile(secrets []string, dir string) map[string]s.Secret {
	res := make(map[string]s.Secret)

	secretBackend := fileBackend{rootPath: dir}
	for _, secretId := range secrets {
		res[secretId] = secretBackend.get(secretId)
	}

	return res
}

func splitVersion(ver string) []string {
	return strings.SplitN(ver, ".", 2)
}
