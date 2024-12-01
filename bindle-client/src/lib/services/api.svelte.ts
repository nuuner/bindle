import { config } from "$lib/config";
import { getAccountId, newAccountId, setAccount } from "$lib/stores/accountStore.client.svelte";
import { addFile, deleteFile } from "$lib/stores/fileStore.svelte";
import { addUploadingFile, removeUploadingFile } from '$lib/stores/uploadStore.svelte';
import type { Account, UploadedFile } from '$lib/types';

const getHeaders = (isJson: boolean = true, accountId?: string) => {
    const headers: Record<string, string> = {
        Authorization: accountId || getAccountId() || "",
    };
    if (isJson) {
        headers['Content-Type'] = 'application/json';
    }
    return headers;
};

export const getFilesFromServer = async () => {
    const response = await fetch(`${config.apiHost}/files`, {
        headers: getHeaders(),
    });
    return response.json();
};

export const getMe = async (accountId?: string): Promise<Account> => {
    try {
        const response = await fetch(`${config.apiHost}/me`, {
            headers: getHeaders(false, accountId),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const meResponse = await response.json();
        if (getAccountId() === meResponse.user.accountId) {
            setAccount(meResponse);
        }
        return meResponse;
    } catch (error) {
        console.error('Failed to fetch account:', error);
        throw error;
    }
};

export const deleteAccount = async () => {
    const response = await fetch(`${config.apiHost}/me`, {
        method: "DELETE",
        headers: getHeaders(),
    });
    const json = await response.json();
    newAccountId();
    return json;
};

export const updateFile = async (file: UploadedFile) => {
    const response = await fetch(`${config.apiHost}/file`, {
        method: "PUT",
        headers: getHeaders(true),
        body: JSON.stringify(file),
    });
    return response.json();
};

export const uploadFile = async (file: File) => {
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
        getMe();
    }
};

export const eraseFile = async (fileId: string) => {
    const response = await fetch(`${config.apiHost}/file/${fileId}`, {
        method: "DELETE",
        headers: getHeaders(),
    });
    deleteFile(fileId);
    getMe();
    return response.json();
};
