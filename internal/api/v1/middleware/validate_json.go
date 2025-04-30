// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	cuejson "cuelang.org/go/encoding/json"
)

type validate struct {
	next   http.Handler
	schema cue.Value
}

func (v validate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := LoggerFrom(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	r.Body.Close()

	err = cuejson.Validate(body, v.schema)
	if err != nil {
		cueerrors := errors.Errors(err)
		entityErrors := make([]string, len(cueerrors))

		for i, cueerr := range cueerrors {
			logger.Debug("cue error validating body", "cuerr", cueerr.Error())

			entityErrors[i] = cueerr.Error()
		}

		unprocessableEntityErrors := types.UnprocessableEntityErrors{
			Errors: entityErrors,
		}

		b, err := json.Marshal(&unprocessableEntityErrors)
		if err != nil {
			logger.Error("error marshalling unprocessable entity errors", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusUnprocessableEntity)

		_, err = w.Write(b)
		if err != nil {
			logger.Error("error writing body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	v.next.ServeHTTP(w, r)
}

type ValidateJSON struct {
	instance *cue.Value
}

func (j *ValidateJSON) WithSchema(s string) func(http.Handler) http.Handler {
	schema := j.instance.LookupPath(cue.ParsePath(s))
	if schema.Err() != nil {
		panic(schema.Err())
	}

	fn := func(next http.Handler) http.Handler {
		return validate{schema: schema, next: next}
	}

	return fn
}

//go:embed validate_json.cue
var s string

func NewValidateJSON() (*ValidateJSON, error) {
	source := load.FromString(s)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	instances := load.Instances([]string{}, &load.Config{
		Package: "middleware",
		Overlay: map[string]load.Source{
			path.Join(wd, "validate_json.cue"): source,
		},
	})

	for _, instance := range instances {
		if instance.Err != nil {
			return nil, instance.Err
		}
	}

	cuectx := cuecontext.New()

	instance := cuectx.BuildInstance(instances[0])
	if instance.Err() != nil {
		return nil, instance.Err()
	}

	j := ValidateJSON{
		instance: &instance,
	}

	return &j, nil
}
