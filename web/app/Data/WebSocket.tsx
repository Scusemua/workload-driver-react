export interface WebSocketMessage {
    op: string;
    payload: any;
}

export interface ErrorMessage {
    errorName: string;
    errorMessage: string;
}
