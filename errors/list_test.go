// Copyright 2022 Mineiros GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors_test

import (
	stderrors "errors"
	stdfmt "fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/madlambda/spells/assert"
	"github.com/mineiros-io/terramate/errors"
)

func TestEmptyErrorListReturnsEmptyErrors(t *testing.T) {
	e := errors.L()
	errs := e.Errors()

	assert.EqualInts(t, 0, len(errs))
}

func TestErrorListReturnsAllErrors(t *testing.T) {
	// This test was updated to reflect the changed API of errors.List.Errors().
	// Now it doesn't ignore errors anymore.

	e := errors.L()

	assert.EqualInts(t, 0, len(e.Errors()))

	notIgnored := stderrors.New("not ignored")
	e.Append(E("one"))
	e.Append(stdfmt.Errorf("wrapped: %w", E("two")))
	e.Append(notIgnored)
	e.Append(E("three"))

	errs := e.Errors()

	assert.EqualInts(t, 4, len(errs))
	assert.IsError(t, errs[0], E("one"))
	assert.IsError(t, errs[1], E("two"))
	assert.IsError(t, errs[2], notIgnored)
	assert.IsError(t, errs[3], E("three"))
}

func TestEmptyErrorListStringRepresentationIsEmpty(t *testing.T) {
	errs := errors.L()
	assert.EqualStrings(t, "", errs.Error())
	assert.EqualStrings(t, "", errs.Detailed())
}

func TestEmptyErrorListAsErrorIsNil(t *testing.T) {
	errs := errors.L()
	err := errs.AsError()
	if err != nil {
		t.Fatalf("got error %v but want nil", err)
	}
}

func TestErrorListIgnoresNilErrors(t *testing.T) {
	errs := errors.L(nil, nil)
	errs.Append(nil)
	err := errs.AsError()
	if err != nil {
		t.Fatalf("got error %v but want nil", err)
	}
}

func TestErrorListFlattensAllDiagnostics(t *testing.T) {
	const (
		detail1 = "error 1"
		detail2 = "error 2"
	)
	var (
		range1 = hcl.Range{
			Filename: "file1.tm",
			Start:    hcl.Pos{Line: 1, Column: 5, Byte: 3},
			End:      hcl.Pos{Line: 1, Column: 10, Byte: 13},
		}

		range2 = hcl.Range{
			Filename: "file2.tm",
			Start:    hcl.Pos{Line: 2, Column: 6, Byte: 4},
			End:      hcl.Pos{Line: 2, Column: 11, Byte: 14},
		}
	)
	diags := hcl.Diagnostics{
		&hcl.Diagnostic{
			Detail:   detail1,
			Severity: hcl.DiagError,
			Subject:  &range1,
		},
		&hcl.Diagnostic{
			Detail:   detail2,
			Severity: hcl.DiagError,
			Subject:  &range2,
		},
	}

	errs := errors.L()
	errs.Append(diags)

	wantErrs := []error{
		errors.E(detail1, range1),
		errors.E(detail2, range2),
	}
	gotErrs := errs.Errors()

	if diff := cmp.Diff(gotErrs, wantErrs); diff != "" {
		t.Fatalf("-(got) +(want):\n%s", diff)
	}
}

func TestErrorListFlattensOtherErrorList(t *testing.T) {
	const (
		kind1 errors.Kind = "kind1"
		kind2 errors.Kind = "kind2"
		kind3 errors.Kind = "kind3"
	)

	error1 := errors.E(kind1)
	error2 := errors.E(kind2)
	error3 := errors.E(kind3)

	errs := errors.L(error1)
	errs.Append(errors.L(error2, error3))

	wantErrs := []error{error1, error2, error3}
	gotErrs := errs.Errors()

	if diff := cmp.Diff(gotErrs, wantErrs); diff != "" {
		t.Fatalf("-(got) +(want):\n%s", diff)
	}
}

func TestErrorListsWithMatchingErrors(t *testing.T) {
	const (
		kind1 errors.Kind = "kind1"
		kind2 errors.Kind = "kind2"
		kind3 errors.Kind = "kind3"
	)

	t.Run("empty list match", func(t *testing.T) {
		assert.IsTrue(t, errors.Is(errors.L(), errors.L()))
	})

	errs1 := errors.L(errors.E(kind1), errors.E(kind2), errors.E(kind3))

	t.Run("exact errors match", func(t *testing.T) {
		errs2 := errors.L(errors.E(kind1), errors.E(kind2), errors.E(kind3))
		assert.IsTrue(t, errors.Is(errs1, errs2))
	})

	t.Run("wrong order doesnt match", func(t *testing.T) {
		errs2 := errors.L(errors.E(kind2), errors.E(kind1), errors.E(kind3))
		assert.IsTrue(t, !errors.Is(errs1, errs2))
	})

	t.Run("partial doesnt match", func(t *testing.T) {
		assert.IsTrue(t, !errors.Is(errs1, errors.L(errors.E(kind1))))
	})
}

func TestErrorListStringDetailedPresentation(t *testing.T) {
	errs := errors.L(E("one"))
	assert.EqualStrings(t, "error list:\n\t-one", errs.Detailed())

	errs.Append(stderrors.New("two"))
	assert.EqualStrings(t, "error list:\n\t-one\n\t-two", errs.Detailed())
}
