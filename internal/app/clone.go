package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// CalcNextCloneJump returns when the next clone jump is available,
// based on the last jump and the Infomorph Synchronizing skill level.
// It returns a zero time when a jump is available now
// and empty when the last jump is unknown.
func CalcNextCloneJump(lastJumpAt optional.Optional[time.Time], infomorphSyncLevel int64, now time.Time) optional.Optional[time.Time] {
	lastJump, ok := lastJumpAt.Value()
	if !ok {
		return optional.Optional[time.Time]{}
	}
	nextJump := lastJump.Add(time.Duration(24-infomorphSyncLevel) * time.Hour)
	if nextJump.Before(now) {
		return optional.New(time.Time{})
	}
	return optional.New(nextJump)
}

type CharacterImplant struct {
	CharacterID int64
	EveType     *EveType
	ID          int64
	SlotNum     int // 0 = unknown
}

type CharacterJumpClone struct {
	CharacterID int64
	ID          int64
	Implants    []*CharacterJumpCloneImplant
	CloneID     int64
	Location    *EveLocationShort
	Name        optional.Optional[string]
	Region      *EntityShort
}

type CharacterJumpClone2 struct {
	Character     *EntityShort
	ImplantsCount int
	ID            int64
	CloneID       int64
	Location      *EveLocation
}

type CharacterJumpCloneImplant struct {
	ID      int64
	EveType *EveType
	SlotNum int // 0 = unknown
}
