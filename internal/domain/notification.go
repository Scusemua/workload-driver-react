package domain

const (
	ErrorNotification   NotificationType = 0
	WarningNotification NotificationType = 1
	InfoNotification    NotificationType = 2
	SuccessNotification NotificationType = 3
)

type NotificationType int32
