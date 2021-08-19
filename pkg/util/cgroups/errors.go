// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package cgroups

import (
	"errors"
	"fmt"
	"strings"
)

type InvalidInputError struct {
	Desc string
}

func (e *InvalidInputError) Error() string {
	return "invalid input: " + e.Desc
}

type ControllerNotFoundError struct {
	Controller string
}

func (e *ControllerNotFoundError) Error() string {
	return "mount point for cgroup controller not found: " + e.Controller
}

func (e *ControllerNotFoundError) Is(target error) bool {
	t, ok := target.(*ControllerNotFoundError)
	if !ok {
		return false
	}
	return e.Controller == t.Controller
}

type wrappedError struct {
	Err error
}

func (e *wrappedError) Error() string {
	return e.Err.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.Err
}

type FileSystemError struct {
	wrappedError
}

type ValueError struct {
	wrappedError
}

// Adapted from https://github.com/kubernetes/apimachinery/blob/master/pkg/util/errors/errors.go
// Copyright 2015 The Kubernetes Authors.

type ErrorList interface {
	error
	Errors() []error
	Is(error) bool
}

type errorList []error

func (e errorList) Error() string {
	sb := strings.Builder{}
	for i, err := range e {
		fmt.Fprintf(&sb, "error %d: '%s'\n", i, err.Error())
	}
	return sb.String()
}

func (e errorList) Is(target error) bool {
	return e.visit(func(err error) bool {
		return errors.Is(err, target)
	})
}

func (e errorList) Errors() []error {
	return []error(e)
}

func (e errorList) visit(f func(err error) bool) bool {
	for _, err := range e {
		switch err := err.(type) {
		case errorList:
			if match := err.visit(f); match {
				return match
			}
		case ErrorList:
			for _, nestedErr := range err.Errors() {
				if match := f(nestedErr); match {
					return match
				}
			}
		default:
			if match := f(err); match {
				return match
			}
		}
	}

	return false
}
