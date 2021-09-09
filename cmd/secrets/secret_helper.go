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

// This executable provides a "read" command to decode secrets. It can be used
// in 2 different ways:
//
// 1) With the "--with-backend-prefixes" option enabled. Each input secret should
// follow this format: "backendPrefix/some/path". The backend prefix indicates
// where to fetch the secrets from. At the moment, we support "file" and
// "kube_secret". The path can mean different things depending on the backend.
// In "file" it's a file system path. In "kube_secret", it follows this format:
// "namespace/name/key".
//
// 2) With the "--with-backend-prefixes" option disabled. The program expect a root
// path in the arguments and input secrets are just paths relative to the root
// one. So for example, if the secret is "my_secret" and the root path is
// "/some/path", the decoded value of the secret will be the contents of
// "/some/path/my_secret". This option was offered before introducing
// "--with-backend-prefixes" and is kept to avoid breaking compatibility.

const (
	backendPrefixesFlag    = "with-backend-prefixes"
	backendPrefixSeparator = "/"
)

func init() {
	cmd := readSecretCmd
	cmd.Flags().Bool(backendPrefixesFlag, false, "Use prefixes to select the secret backend (file, kube_secret)")
	SecretHelperCmd.AddCommand(cmd)
}

// SecretHelperCmd implements secrets backend helper commands
var SecretHelperCmd = &cobra.Command{
	Use:   "secret-helper",
	Short: "Secret management backend helper",
	Long:  ``,
}

var readSecretCmd = &cobra.Command{
	Use:   "read",
	Short: "Read secrets",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1), // 0 when using the backend prefixes option, 1 when reading a file
	RunE: func(cmd *cobra.Command, args []string) error {
		usePrefixes, err := cmd.Flags().GetBool(backendPrefixesFlag)
		if err != nil {
			return err
		}

		dir := ""
		if len(args) == 1 {
			dir = args[0]
		}

		return decodeSecrets(os.Stdin, os.Stdout, dir, usePrefixes, &backendSelector{})
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

func decodeSecrets(r io.Reader, w io.Writer, dir string, usePrefixes bool, selector *backendSelector) error {
	inputSecrets, err := parseInputSecrets(r)
	if err != nil {
		return err
	}

	var decodedSecrets map[string]s.Secret
	if usePrefixes {
		decodedSecrets = readSecretsUsingPrefixes(inputSecrets, selector)
	} else {
		decodedSecrets = readSecretsFromFile(inputSecrets, dir)
	}

	return writeDecodedSecrets(w, decodedSecrets)
}

func parseInputSecrets(r io.Reader) ([]string, error) {
	in, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var request secretsRequest
	err = json.Unmarshal(in, &request)
	if err != nil {
		return nil, errors.New("failed to unmarshal json input")
	}

	version := splitVersion(request.Version)
	compatVersion := splitVersion(s.PayloadVersion)
	if version[0] != compatVersion[0] {
		return nil, fmt.Errorf("incompatible protocol version %q", request.Version)
	}

	if len(request.Secrets) == 0 {
		return nil, errors.New("no secrets listed in input")
	}

	return request.Secrets, nil
}

func writeDecodedSecrets(w io.Writer, resolvedSecrets map[string]s.Secret) error {
	out, err := json.Marshal(resolvedSecrets)
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	return err
}

func readSecretsFromFile(secrets []string, dir string) map[string]s.Secret {
	res := make(map[string]s.Secret)

	secretBackend := fileBackend{rootPath: dir}
	for _, secretID := range secrets {
		res[secretID] = secretBackend.get(secretID)
	}

	return res
}

func readSecretsUsingPrefixes(secrets []string, selector *backendSelector) map[string]s.Secret {
	res := make(map[string]s.Secret)

	for _, secretID := range secrets {
		prefix, id, err := parseSecretWithPrefix(secretID)
		if err != nil {
			res[secretID] = s.Secret{Value: "", ErrorMsg: err.Error()}
			continue
		}

		secretBackend, err := selector.choose(prefix)
		if err != nil {
			res[secretID] = s.Secret{Value: "", ErrorMsg: err.Error()}
			continue
		}

		res[secretID] = secretBackend.get(id)
	}

	return res
}

func parseSecretWithPrefix(secretID string) (prefix backendPrefix, id string, err error) {
	split := strings.SplitN(secretID, backendPrefixSeparator, 2)
	if len(split) != 2 {
		return "", "", errors.New("invalid secret format")
	}

	prefix = backendPrefix(split[0])
	if !prefix.isValid() {
		return "", "", fmt.Errorf("backend not supported")
	}

	id = split[1]
	return prefix, id, nil
}

func splitVersion(ver string) []string {
	return strings.SplitN(ver, ".", 2)
}
