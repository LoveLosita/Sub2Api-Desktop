package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type SchedulerService struct {
	db    *database.DB
	mu    sync.Mutex
	index map[string]int // round-robin index per group+platform
}

func NewSchedulerService(db *database.DB) *SchedulerService {
	return &SchedulerService{
		db:    db,
		index: make(map[string]int),
	}
}

// PickAccount selects the next available account for the given group and platform.
// Uses priority-based round-robin: accounts are sorted by priority, then cycled through.
func (s *SchedulerService) PickAccount(groupID int64, platform string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	as := NewAccountService(s.db)
	accounts, err := as.GetSchedulableForGroup(groupID)
	if err != nil {
		return nil, err
	}

	// Filter by platform
	var filtered []model.Account
	for _, a := range accounts {
		if a.Platform == platform {
			filtered = append(filtered, a)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no available %s accounts in group %d", platform, groupID)
	}

	key := fmt.Sprintf("%d:%s", groupID, platform)
	idx := s.index[key] % len(filtered)

	// Simple failover: try starting from idx, then wrap around
	for i := 0; i < len(filtered); i++ {
		candidate := filtered[(idx+i)%len(filtered)]
		if candidate.Status == "active" && candidate.Schedulable {
			s.index[key] = (idx + i + 1) % len(filtered)
			return &candidate, nil
		}
	}

	// Fallback: pick any account
	picked := filtered[0]
	s.index[key] = (idx + 1) % len(filtered)
	return &picked, nil
}

// PickAccountForPlatform selects an account without a group (uses any active account of the platform).
func (s *SchedulerService) PickAccountForPlatform(platform string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	as := NewAccountService(s.db)
	all, err := as.List()
	if err != nil {
		return nil, err
	}

	var filtered []model.Account
	now := time.Now()
	for _, a := range all {
		if a.Platform != platform || a.Status != "active" || !a.Schedulable {
			continue
		}
		if a.RateLimitResetAt != nil && now.Before(*a.RateLimitResetAt) {
			continue
		}
		if a.OverloadUntil != nil && now.Before(*a.OverloadUntil) {
			continue
		}
		filtered = append(filtered, a)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no available %s accounts", platform)
	}

	key := "global:" + platform
	idx := s.index[key] % len(filtered)
	s.index[key] = (idx + 1) % len(filtered)

	// Add small random jitter to avoid thundering herd
	if len(filtered) > 1 {
		jitter := rand.Intn(min(3, len(filtered)))
		idx = (idx + jitter) % len(filtered)
	}

	return &filtered[idx], nil
}
