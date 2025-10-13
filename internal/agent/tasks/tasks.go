package tasks

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

type ExecutableTask interface {
	Execute(ctx context.Context) error
}

type GeneralTask struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Payload string `json:"payload"`
}

func (g *GeneralTask) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(g)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal task")
	}
	return string(jsonBytes), nil
}

func (g *GeneralTask) FromJSON(data []byte) error {
	gt := &GeneralTask{}
	err := json.Unmarshal(data, gt)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal task")
	}

	g.ID = gt.ID
	g.Name = gt.Name
	g.Payload = gt.Payload

	return nil
}
