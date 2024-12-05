import { config } from "$lib/config";
import { getAccount, getAccountId } from "$lib/stores/accountStore.client.svelte";
import { addFile, deleteFile as removeFileFromStore } from "$lib/stores/fileStore.svelte";
import { addUploadingFile, removeUploadingFile } from '$lib/stores/uploadStore.svelte';
import { setError } from "$lib/stores/errorStore.svelte";
import type { UploadedFile } from '$lib/types';
import { accountService } from "./accountService";

export const getHeaders = (isJson: boolean = true, accountId?: string) => {
    const headers: Record<string, string> = {};
    if (isJson) {
        headers['Content-Type'] = 'application/json';
    }
    headers['Authorization'] = accountId || getAccountId() || "";
    return headers;
};

export const fileService = {
    async getFiles() {
        const response = await fetch(`${config.apiHost}/files`, {
            headers: getHeaders(),
        });
        return response.json();
    },

    async updateFile(file: UploadedFile) {
        const response = await fetch(`${config.apiHost}/file`, {
            method: "PUT",
            headers: getHeaders(true),
            body: JSON.stringify(file),
        });
        return response.json();
    },

    async uploadFile(file: File) {
        const account = getAccount();

        if (!account) {
            throw new Error("Account not found");
        }

        if (file.size > account.maxFileSizeBytes) {
            setError(`File is too large. Max file size is ${Math.round(account.maxFileSizeBytes / 1000 / 1000)}MB.`);
            return;
        }

        if (account && account.uploadLimitBytes && account.uploadLimitBytes < (file.size + account.uploadedBytes)) {
            setError(`Upload limit exceeded. You may only upload up to ${Math.round(account.uploadLimitBytes / 1000 / 1000)}MB per day. Wait or delete some files.`);
            return;
        }

        const uploadingId = addUploadingFile(file);
        try {
            const formData = new FormData();
            formData.append("file", file);

            const response = await fetch(`${config.apiHost}/file`, {
                method: "POST",
                headers: getHeaders(false),
                body: formData,
            });
            const uploadedFile = await response.json();
            addFile(uploadedFile);
            return uploadedFile;
        } finally {
            removeUploadingFile(uploadingId);
            accountService.getMe();
        }
    },

    async deleteFile(fileId: string) {
        const response = await fetch(`${config.apiHost}/file/${fileId}`, {
            method: "DELETE",
            headers: getHeaders(),
        });
        removeFileFromStore(fileId);
        accountService.getMe();
        return response.json();
    }
}; 