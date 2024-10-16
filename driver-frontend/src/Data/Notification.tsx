export interface Notification {
    id: string;
    title: string;
    message: string;
    notificationType: 0 | 1 | 2 | 3;
    panicked: boolean;
}
