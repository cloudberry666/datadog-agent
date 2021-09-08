// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build secrets

package secrets

import (
	"errors"
	"fmt"
	s "github.com/DataDog/datadog-agent/pkg/secrets"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	maxSecretFileSize = 8192
)

type fileBackend struct {
	rootPath string
}

func (fb *fileBackend) get(secretID string) s.Secret {
	return readSecret(filepath.Join(fb.rootPath, secretID))
}

func readSecret(path string) s.Secret {
	value, err := readSecretFile(path)
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	return s.Secret{Value: value, ErrorMsg: errMsg}
}

func readSecretFile(path string) (string, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("secret does not exist")
		}
		return "", err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		// Ensure that the symlink is in the same dir
		target, err := os.Readlink(path)
		if err != nil {
			return "", fmt.Errorf("failed to read symlink target: %v", err)
		}

		dir := filepath.Dir(path)
		if !filepath.IsAbs(target) {
			target, err = filepath.Abs(filepath.Join(dir, target))
			if err != nil {
				return "", fmt.Errorf("failed to resolve symlink absolute path: %v", err)
			}
		}

		dirAbs, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path of directory: %v", err)
		}

		if !filepath.HasPrefix(target, dirAbs) {
			return "", fmt.Errorf("not following symlink %q outside of %q", target, dir)
		}
	}
	fi, err = os.Stat(path)
	if err != nil {
		return "", err
	}

	if fi.Size() > maxSecretFileSize {
		return "", errors.New("secret exceeds max allowed size")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
