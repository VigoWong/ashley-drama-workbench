package llm

import (
	"context"
	"fmt"
)

type Mock struct{ fixtures map[string]string }

func NewMock() *Mock { return &Mock{fixtures: map[string]string{}} }

func (m *Mock) Register(stage, jsonOut string) { m.fixtures[stage] = jsonOut }

func (m *Mock) GenerateJSON(_ context.Context, stage, _ string, _ map[string]any) ([]byte, error) {
	v, ok := m.fixtures[stage]
	if !ok {
		return nil, fmt.Errorf("mock: no fixture registered for stage %q", stage)
	}
	return []byte(v), nil
}
