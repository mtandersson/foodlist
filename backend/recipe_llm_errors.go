package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// RecipeLLMErrorKind classifies provider failures without exposing upstream
// response bodies or credential details to HTTP callers or logs.
type RecipeLLMErrorKind string

const (
	LLMErrorAuth     RecipeLLMErrorKind = "auth"
	LLMErrorQuota    RecipeLLMErrorKind = "quota"
	LLMErrorTimeout  RecipeLLMErrorKind = "timeout"
	LLMErrorUpstream RecipeLLMErrorKind = "upstream"
	LLMErrorInvalid  RecipeLLMErrorKind = "invalid_response"
	LLMErrorConfig   RecipeLLMErrorKind = "config"
)

type RecipeLLMError struct {
	Kind              RecipeLLMErrorKind
	RetryAfterSeconds int
	cause             error
}

func (e *RecipeLLMError) Error() string {
	if e == nil {
		return "recipe llm error"
	}
	return fmt.Sprintf("recipe llm %s", e.Kind)
}

func (e *RecipeLLMError) Unwrap() error { return e.cause }

func newRecipeLLMError(kind RecipeLLMErrorKind, cause error) error {
	return &RecipeLLMError{Kind: kind, cause: cause}
}

func newRecipeLLMQuotaError(retryAfter string) error {
	seconds := 0
	if n, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && n > 0 && n <= 24*60*60 {
		seconds = n
	}
	return &RecipeLLMError{Kind: LLMErrorQuota, RetryAfterSeconds: seconds}
}

func asRecipeLLMError(err error) (*RecipeLLMError, bool) {
	var target *RecipeLLMError
	ok := errors.As(err, &target)
	return target, ok
}
