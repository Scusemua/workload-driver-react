package domain

const (
	ErrorNotification   NotificationType = 0
	WarningNotification NotificationType = 1
	InfoNotfication     NotificationType = 2
	SuccessNotification NotificationType = 3
)

type NotificationType int32
