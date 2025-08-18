import { accountService } from "./accountService";
import { getAccountId } from "$lib/stores/accountStore.client.svelte";

class SyncService {
    private pollingInterval: number | null = null;
    private isPollingActive = false;
    private readonly POLLING_INTERVAL = 5000;

    constructor() {
        if (typeof document !== "undefined") {
            document.addEventListener("visibilitychange", this.handleVisibilityChange);
        }
    }

    private handleVisibilityChange = () => {
        if (document.hidden) {
            this.stopPolling();
        } else {
            if (getAccountId()) {
                this.startPolling();
            }
        }
    };

    startPolling() {
        if (this.isPollingActive || !getAccountId()) return;

        this.isPollingActive = true;
        
        if (!document.hidden) {
            this.pollingInterval = window.setInterval(async () => {
                if (!document.hidden && getAccountId()) {
                    try {
                        await accountService.getMe();
                    } catch (error) {
                        console.error("Polling error:", error);
                    }
                }
            }, this.POLLING_INTERVAL);
        }
    }

    stopPolling() {
        if (this.pollingInterval) {
            clearInterval(this.pollingInterval);
            this.pollingInterval = null;
        }
        this.isPollingActive = false;
    }

    cleanup() {
        this.stopPolling();
        
        if (typeof document !== "undefined") {
            document.removeEventListener("visibilitychange", this.handleVisibilityChange);
        }
    }
}

export const syncService = new SyncService();