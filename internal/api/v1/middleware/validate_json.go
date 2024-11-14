package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

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

func NewValidateJSON(dir string) (*ValidateJSON, error) {
	instances := load.Instances([]string{}, &load.Config{
		Dir:     dir,
		Package: "middleware",
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
