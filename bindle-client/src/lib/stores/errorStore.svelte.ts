let error = $state<string | null>(null);
let timeout = $state<number | undefined>(undefined);

export function getTimeout() {
    return timeout;
}
export function getError() {
    return error;
}

export function clearError() {
    error = null;
    timeout = undefined;
}
export function setError(message: string) {
    error = message;
    timeout = 5000;
}
