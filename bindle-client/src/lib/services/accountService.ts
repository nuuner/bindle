import { config } from "$lib/config";
import type { Account } from "$lib/types";
import { getAccountId, setAccount, setAccountId } from "$lib/stores/accountStore.client.svelte";
import { setFiles } from "$lib/stores/fileStore.svelte";

export const getHeaders = (isJson: boolean = true, accountId?: string) => {
    const headers: Record<string, string> = {
        Authorization: accountId || getAccountId() || "",
    };
    if (isJson) {
        headers['Content-Type'] = 'application/json';
    }
    return headers;
};

export const accountService = {
    async getMe(accountId?: string): Promise<Account> {
        try {
            const response = await fetch(`${config.apiHost}/me`, {
                headers: getHeaders(false, accountId),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const meResponse = await response.json();

            setAccount(meResponse);
            setFiles(meResponse.user.files);

            return meResponse;
        } catch (error) {
            console.error('Failed to fetch account:', error);
            throw error;
        }
    },

    async getMeWithoutAccountId(): Promise<Account> {
        const newAccount = await this.getMe(undefined);
        setAccountId(newAccount.user.accountId);
        setAccount(newAccount);
        setFiles(newAccount.user.files);
        return newAccount;
    },

    async deleteAccount() {
        const response = await fetch(`${config.apiHost}/me`, {
            method: "DELETE",
            headers: getHeaders(),
        });
        const json = await response.json();
        setAccountId(undefined);
        await this.initializeAccount();
    },

    async initializeAccount() {
        const idFromLocalStorage = localStorage.getItem("bindle.accountId");
        if (idFromLocalStorage) {
            setAccountId(idFromLocalStorage);
            await this.getMe(idFromLocalStorage);
        } else {
            await this.getMeWithoutAccountId();
        }
    }
}; 