package adminapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
)

func (a *AdminAPI) GetAllVisitors() ([]*entity.Visitor, error) {
	visitors, err := a.VisitorRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get visitors: %w", err)
	}

	for _, v := range visitors {
		keys, err := a.VisitorRepo.FindKeysByVisitorId(v.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get keys for visitor %d: %w", v.Id, err)
		}
		v.Keys = keys
	}

	return visitors, nil
}

func (a *AdminAPI) GetVisitor(id int32) (*entity.Visitor, error) {
	visitor, err := a.VisitorRepo.FindById(id)
	if err != nil {
		return nil, fmt.Errorf("failed to find visitor: %w", err)
	}
	if visitor.Id == 0 {
		return nil, fmt.Errorf("visitor not found")
	}

	keys, err := a.VisitorRepo.FindKeysByVisitorId(visitor.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	visitor.Keys = keys

	return visitor, nil
}

func (a *AdminAPI) CreateVisitor(name, surname string, isStudent bool, grade *int, email string, keys []string) (*entity.Visitor, error) {
	visitor := &entity.Visitor{
		Name:      name,
		Surname:   surname,
		Email:     email,
		IsStudent: isStudent,
		Grade:     grade,
		UpdatedAt: time.Now(),
	}

	if err := a.VisitorRepo.SaveVisitor(visitor); err != nil {
		return nil, fmt.Errorf("failed to create visitor: %w", err)
	}

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		key = strings.ToUpper(key)
		if err := a.VisitorRepo.AddKeyToVisitor(visitor, key); err != nil {
			return nil, fmt.Errorf("failed to add key %q: %w", key, err)
		}
	}

	savedKeys, err := a.VisitorRepo.FindKeysByVisitorId(visitor.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	visitor.Keys = savedKeys

	return visitor, nil
}

func (a *AdminAPI) UpdateVisitor(id int32, name, surname string, isStudent bool, grade *int, email string, keys []string) (*entity.Visitor, error) {
	visitor, err := a.VisitorRepo.FindById(id)
	if err != nil {
		return nil, fmt.Errorf("failed to find visitor: %w", err)
	}
	if visitor.Id == 0 {
		return nil, fmt.Errorf("visitor not found")
	}

	visitor.Name = name
	visitor.Surname = surname
	visitor.Email = email
	visitor.IsStudent = isStudent
	visitor.Grade = grade
	visitor.UpdatedAt = time.Now()

	if err := a.VisitorRepo.SaveVisitor(visitor); err != nil {
		return nil, fmt.Errorf("failed to update visitor: %w", err)
	}

	// Sync keys: remove old ones not in new set, add new ones not in old set
	existingKeys, err := a.VisitorRepo.FindKeysByVisitorId(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing keys: %w", err)
	}

	newKeysMap := make(map[string]bool)
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			newKeysMap[strings.ToUpper(k)] = true
		}
	}

	existingKeysMap := make(map[string]bool)
	for _, k := range existingKeys {
		existingKeysMap[k] = true
	}

	// Remove keys that are no longer in the set
	for _, k := range existingKeys {
		if !newKeysMap[k] {
			if err := a.VisitorRepo.RemoveKeyFromVisitor(id, k); err != nil {
				return nil, fmt.Errorf("failed to remove key %q: %w", k, err)
			}
		}
	}

	// Add keys that are new
	for k := range newKeysMap {
		if !existingKeysMap[k] {
			if err := a.VisitorRepo.AddKeyToVisitor(visitor, k); err != nil {
				return nil, fmt.Errorf("failed to add key %q: %w", k, err)
			}
		}
	}

	savedKeys, err := a.VisitorRepo.FindKeysByVisitorId(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	visitor.Keys = savedKeys

	return visitor, nil
}

func (a *AdminAPI) UploadVisitorImage(id int32, filename string, data []byte) (*entity.Visitor, error) {
	visitor, err := a.VisitorRepo.FindById(id)
	if err != nil {
		return nil, fmt.Errorf("failed to find visitor: %w", err)
	}
	if visitor.Id == 0 {
		return nil, fmt.Errorf("visitor not found")
	}

	dir := a.Config.VisitorImageDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	ext := filepath.Ext(filename)
	storedFilename := fmt.Sprintf("%d%s", id, ext)
	filePath := filepath.Join(dir, storedFilename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write image file: %w", err)
	}

	imageURL := a.Config.VisitorImageURLPrefix + "/" + storedFilename
	if err := a.VisitorRepo.UpdateVisitorImage(id, imageURL); err != nil {
		return nil, fmt.Errorf("failed to update visitor image: %w", err)
	}

	visitor.Image = imageURL
	return visitor, nil
}

func (a *AdminAPI) AddVisitorKey(visitorID int32, key string) error {
	visitor, err := a.VisitorRepo.FindById(visitorID)
	if err != nil {
		return fmt.Errorf("failed to find visitor: %w", err)
	}
	if visitor.Id == 0 {
		return fmt.Errorf("visitor not found")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if err := a.VisitorRepo.AddKeyToVisitor(visitor, key); err != nil {
		return fmt.Errorf("failed to add key: %w", err)
	}

	return nil
}

func (a *AdminAPI) RemoveVisitorKey(visitorID int32, key string) error {
	visitor, err := a.VisitorRepo.FindById(visitorID)
	if err != nil {
		return fmt.Errorf("failed to find visitor: %w", err)
	}
	if visitor.Id == 0 {
		return fmt.Errorf("visitor not found")
	}

	if err := a.VisitorRepo.RemoveKeyFromVisitor(visitorID, key); err != nil {
		return fmt.Errorf("failed to remove key: %w", err)
	}

	return nil
}

func (a *AdminAPI) DeleteVisitor(id int32) error {
	if err := a.VisitorRepo.DeleteVisitor(id); err != nil {
		return fmt.Errorf("failed to delete visitor: %w", err)
	}
	return nil
}
