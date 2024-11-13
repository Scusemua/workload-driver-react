package domain

const (
	ErrorNotification   NotificationType = 0
	WarningNotification NotificationType = 1
	InfoNotification    NotificationType = 2
	SuccessNotification NotificationType = 3
)

var (
	NotificationTypeNames = []string{"Error", "Warning", "Info", "Success"}
)

type NotificationType int32

func (t NotificationType) String() string {
	return NotificationTypeNames[int(t)]
}

func (t NotificationType) Int() int {
	return int(t)
}

func (t NotificationType) Int32() int32 {
	return int32(t)
}
