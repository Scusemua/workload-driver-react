export interface Notification {
    title: string;
    message: string;
    notificationType: 0 | 1 | 2 | 3;
    panicked: boolean;
}
