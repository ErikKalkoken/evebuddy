package logic

// Special mail label IDs
const (
	LabelAll      = 1<<31 - 1
	LabelNone     = 0
	LabelInbox    = 1
	LabelSent     = 2
	LabelCorp     = 4
	LabelAlliance = 8
)

type MailLabel struct {
	ID          uint64
	Character   Character
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int32
}
