package properties

import (
	"context"
	"fmt"
)

func (s *service) UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
	if err := requireAdminActor(input.Actor); err != nil {
		return UpdatePropertyResult{}, err
	}

	if input.AgentID != nil {
		if _, err := s.repository.GetAgentByID(ctx, *input.AgentID); err != nil {
			return UpdatePropertyResult{}, ValidationError{Message: fmt.Sprintf("agent_id %d is invalid", *input.AgentID)}
		}
	}

	res, err := s.repository.UpdateProperty(ctx, propertyUUID, input)
	if err != nil {
		return UpdatePropertyResult{}, fmt.Errorf("update property: %w", err)
	}

	return res, nil
}
