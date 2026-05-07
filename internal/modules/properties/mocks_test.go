package properties

import "context"

// Mock PropertyRepository implementation for service tests
type mockPropertyRepository struct {
	getModalityNameFunc        func(ctx context.Context, modalityID int32) (string, error)
	getAllowedPeriodsFunc      func(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error)
	getPropertySubtypeFunc     func(ctx context.Context, propertyTypeID int32) (string, error)
	getClauseValueTypesFunc    func(ctx context.Context, clauseIDs []int32) (map[int32]int32, error)
	createPropertyFunc         func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
	listPropertiesFunc         func(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error)
	getPropertyClausesFunc     func(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error)
	updatePropertyClausesFunc  func(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error
	getPropertyPhotosFunc      func(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error)
	updatePropertyPhotosFunc   func(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error
	getPropertyServicesFunc    func(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error)
	updatePropertyServicesFunc func(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error
	getPropertyPricesFunc      func(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error)
	updatePropertyPricesFunc   func(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error
	getPropertyFunc            func(ctx context.Context, propertyUUID string) (GetPropertyResult, error)
	getFullPropertyFunc        func(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error)
	updatePropertyFunc         func(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error)
	getPropertyStorageKeysFunc func(ctx context.Context, propertyID int32) ([]string, error)
	deletePropertyFunc         func(ctx context.Context, propertyID int32, changedByUserID int32) error

	// CU-18
	getPropertyOwnerByUUIDFunc   func(ctx context.Context, propertyUUID string) (int32, error)
	listPropertyStatusHistoryFunc func(ctx context.Context, propertyUUID string) ([]PropertyStatusHistoryData, error)
}

func (m *mockPropertyRepository) GetPropertyOwnerByUUID(ctx context.Context, propertyUUID string) (int32, error) {
	if m.getPropertyOwnerByUUIDFunc != nil {
		return m.getPropertyOwnerByUUIDFunc(ctx, propertyUUID)
	}
	return 0, nil
}

func (m *mockPropertyRepository) ListPropertyStatusHistory(ctx context.Context, propertyUUID string) ([]PropertyStatusHistoryData, error) {
	if m.listPropertyStatusHistoryFunc != nil {
		return m.listPropertyStatusHistoryFunc(ctx, propertyUUID)
	}
	return nil, nil
}

func (m *mockPropertyRepository) GetModalityName(ctx context.Context, modalityID int32) (string, error) {
	if m.getModalityNameFunc != nil {
		return m.getModalityNameFunc(ctx, modalityID)
	}
	return "", nil
}

func (m *mockPropertyRepository) GetAllowedPeriods(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
	if m.getAllowedPeriodsFunc != nil {
		return m.getAllowedPeriodsFunc(ctx, propertyTypeID)
	}
	return make(map[int32]struct{}), nil
}

func (m *mockPropertyRepository) GetPropertySubtype(ctx context.Context, propertyTypeID int32) (string, error) {
	if m.getPropertySubtypeFunc != nil {
		return m.getPropertySubtypeFunc(ctx, propertyTypeID)
	}
	return "", nil
}

func (m *mockPropertyRepository) GetClauseValueTypes(ctx context.Context, clauseIDs []int32) (map[int32]int32, error) {
	if m.getClauseValueTypesFunc != nil {
		return m.getClauseValueTypesFunc(ctx, clauseIDs)
	}
	return make(map[int32]int32), nil
}

func (m *mockPropertyRepository) CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	if m.createPropertyFunc != nil {
		return m.createPropertyFunc(ctx, input)
	}
	return CreatePropertyResult{}, nil
}

func (m *mockPropertyRepository) ListProperties(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
	if m.listPropertiesFunc != nil {
		return m.listPropertiesFunc(ctx, input)
	}
	return nil, 0, nil
}

func (m *mockPropertyRepository) GetPropertyClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error) {
	if m.getPropertyClausesFunc != nil {
		return m.getPropertyClausesFunc(ctx, propertyUUID)
	}
	return GetPropertyClausesResult{}, nil
}

func (m *mockPropertyRepository) UpdatePropertyClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error {
	if m.updatePropertyClausesFunc != nil {
		return m.updatePropertyClausesFunc(ctx, propertyUUID, input)
	}
	return nil
}

func (m *mockPropertyRepository) GetPropertyPhotos(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error) {
	if m.getPropertyPhotosFunc != nil {
		return m.getPropertyPhotosFunc(ctx, propertyUUID)
	}
	return GetPropertyPhotosResult{}, nil
}

func (m *mockPropertyRepository) UpdatePropertyPhotos(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error {
	if m.updatePropertyPhotosFunc != nil {
		return m.updatePropertyPhotosFunc(ctx, propertyUUID, input)
	}
	return nil
}

func (m *mockPropertyRepository) GetPropertyServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error) {
	if m.getPropertyServicesFunc != nil {
		return m.getPropertyServicesFunc(ctx, propertyUUID)
	}
	return GetPropertyServicesResult{}, nil
}

func (m *mockPropertyRepository) UpdatePropertyServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error {
	if m.updatePropertyServicesFunc != nil {
		return m.updatePropertyServicesFunc(ctx, propertyUUID, input)
	}
	return nil
}

func (m *mockPropertyRepository) GetPropertyPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error) {
	if m.getPropertyPricesFunc != nil {
		return m.getPropertyPricesFunc(ctx, propertyUUID)
	}
	return GetPropertyPricesResult{}, nil
}

func (m *mockPropertyRepository) UpdatePropertyPrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error {
	if m.updatePropertyPricesFunc != nil {
		return m.updatePropertyPricesFunc(ctx, propertyUUID, input)
	}
	return nil
}

func (m *mockPropertyRepository) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	if m.getPropertyFunc != nil {
		return m.getPropertyFunc(ctx, propertyUUID)
	}
	return GetPropertyResult{}, nil
}

func (m *mockPropertyRepository) GetFullProperty(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error) {
	if m.getFullPropertyFunc != nil {
		return m.getFullPropertyFunc(ctx, propertyUUID)
	}
	return GetPropertyFullResult{}, nil
}

func (m *mockPropertyRepository) UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
	if m.updatePropertyFunc != nil {
		return m.updatePropertyFunc(ctx, propertyUUID, input)
	}
	return UpdatePropertyResult{}, nil
}

func (m *mockPropertyRepository) GetPropertyStorageKeys(ctx context.Context, propertyID int32) ([]string, error) {
	if m.getPropertyStorageKeysFunc != nil {
		return m.getPropertyStorageKeysFunc(ctx, propertyID)
	}
	return nil, nil
}

func (m *mockPropertyRepository) DeleteProperty(ctx context.Context, propertyID int32, changedByUserID int32) error {
	if m.deletePropertyFunc != nil {
		return m.deletePropertyFunc(ctx, propertyID, changedByUserID)
	}
	return nil
}

// Mock storage client
type mockPropertyPhotoStorage struct {
	deleteFunc func(ctx context.Context, storageKey string) error
}

func (m *mockPropertyPhotoStorage) Delete(ctx context.Context, storageKey string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, storageKey)
	}
	return nil
}

func ptrBool(v bool) *bool {
	return &v
}

func ptrInt32(v int32) *int32 {
	return &v
}

func ptrString(v string) *string {
	return &v
}

func ptrInt16(v int16) *int16 {
	return &v
}

func ptrFloat64(v float64) *float64 {
	return &v
}
