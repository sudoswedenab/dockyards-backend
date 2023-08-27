package v1

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type HelmValues map[string]any

func (v *HelmValues) Scan(source any) error {
	switch source := source.(type) {
	case []byte:
		err := json.Unmarshal(source, &v)
		if err != nil {
			return nil
		}
	default:
		return fmt.Errorf("cannot scan helm values of type %T", source)
	}

	return nil
}

func (v HelmValues) Value() (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type CredentialData map[string]string

func (d *CredentialData) Scan(value any) error {
	switch value := value.(type) {
	case []byte:
		err := json.Unmarshal(value, &d)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported value type")
	}

	return nil
}

func (d CredentialData) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return b, nil
}
