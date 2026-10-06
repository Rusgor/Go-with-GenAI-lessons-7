package repository

import (
	"errors"
	"sort"
	"sync"
	"time"

	"homework/internal/model"
)

// DeviceStore describes the storage operations needed by the device handlers.
type DeviceStore interface {
	Create(device model.Device) (model.Device, error)
	Get(id uint32) (model.Device, bool)
	Update(id uint32, device model.Device) (model.Device, bool)
	Delete(id uint32) bool
	List() []model.Device
}

var ErrIDExhausted = errors.New("device ID space exhausted")

const maxDeviceID = uint64(^uint32(0))

// MemoryDeviceStore stores devices in memory for one router instance.
type MemoryDeviceStore struct {
	mu      sync.RWMutex
	devices map[uint32]model.Device
	nextID  uint64
}

// NewMemoryDeviceStore returns an empty device store with IDs starting at 1.
func NewMemoryDeviceStore() *MemoryDeviceStore {
	return &MemoryDeviceStore{
		devices: make(map[uint32]model.Device),
		nextID:  1,
	}
}

// Create stores a device, assigns its ID and timestamps, and returns a copy.
func (s *MemoryDeviceStore) Create(device model.Device) (model.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.nextID > maxDeviceID {
		return model.Device{}, ErrIDExhausted
	}

	now := time.Now().UTC()
	device.ID = uint32(s.nextID)
	device.CreatedAt = now
	device.UpdatedAt = now
	s.nextID++

	device = cloneDevice(device)
	s.devices[device.ID] = device
	return cloneDevice(device), nil
}

// Get returns a copy of the device with the given ID.
func (s *MemoryDeviceStore) Get(id uint32) (model.Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.devices[id]
	if !ok {
		return model.Device{}, false
	}
	return cloneDevice(device), true
}

// Update replaces an existing device while preserving its ID and creation time.
func (s *MemoryDeviceStore) Update(id uint32, device model.Device) (model.Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.devices[id]
	if !ok {
		return model.Device{}, false
	}

	device.ID = id
	device.CreatedAt = existing.CreatedAt
	device.UpdatedAt = time.Now().UTC()
	if !device.UpdatedAt.After(existing.UpdatedAt) {
		device.UpdatedAt = existing.UpdatedAt.Add(time.Nanosecond)
	}

	device = cloneDevice(device)
	s.devices[id] = device
	return cloneDevice(device), true
}

// Delete removes a device and reports whether it existed.
func (s *MemoryDeviceStore) Delete(id uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.devices[id]; !ok {
		return false
	}
	delete(s.devices, id)
	return true
}

// List returns an ID-sorted snapshot of all devices.
func (s *MemoryDeviceStore) List() []model.Device {
	s.mu.RLock()
	devices := make([]model.Device, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, cloneDevice(device))
	}
	s.mu.RUnlock()

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})
	return devices
}

func cloneDevice(device model.Device) model.Device {
	if device.WarrantyMonths != nil {
		value := *device.WarrantyMonths
		device.WarrantyMonths = &value
	}
	if device.Comment != nil {
		value := *device.Comment
		device.Comment = &value
	}
	return device
}
