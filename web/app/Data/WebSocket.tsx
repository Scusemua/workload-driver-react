export interface WebSocketMessage {
    op: string;
    payload: any;
}

export interface Notification {
    title: string;
    message: string;
    notificationType: number;
    panicked: boolean;
}
