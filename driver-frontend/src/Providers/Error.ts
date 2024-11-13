export class RefreshError extends Error {
    public response: Response;
    public statusCode: number;
    public statusText: string;

    public constructor(resp: Response) {
        super(`HTTP ${resp.status} ${resp.statusText}`);

        this.response = resp;
        this.statusCode = resp.status;
        this.statusText = resp.statusText;
    }
}
