package authorization

type RoleBindingService struct{}

func NewRoleBindingService() *RoleBindingService {
	return &RoleBindingService{}
}

func (s *RoleBindingService) PrepareRoleIDs(roleIDs []uint, roles []*Role) ([]uint, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	requested := uniqueRoleIDs(roleIDs)
	found := make(map[uint]struct{}, len(roles))
	for _, role := range roles {
		if role == nil {
			continue
		}
		found[role.GetID()] = struct{}{}
	}

	for _, roleID := range requested {
		if _, ok := found[roleID]; !ok {
			return nil, ErrRoleNotFound
		}
	}
	return requested, nil
}

func uniqueRoleIDs(roleIDs []uint) []uint {
	seen := make(map[uint]struct{}, len(roleIDs))
	result := make([]uint, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if _, ok := seen[roleID]; ok {
			continue
		}
		seen[roleID] = struct{}{}
		result = append(result, roleID)
	}
	return result
}
