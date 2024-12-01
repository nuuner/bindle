import { config } from "$lib/config";
import { getAccountId, setAccount } from "$lib/stores/accountStore.client.svelte";
import { addFile, deleteFile } from "$lib/stores/fileStore.svelte";
import { addUploadingFile, removeUploadingFile } from '$lib/stores/uploadStore.svelte';
import type { Account, UploadedFile } from '$lib/types';

const getHeaders = (isJson: boolean = true) => {
    const headers: Record<string, string> = {
        Authorization: getAccountId() ?? "",
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

export const getMe = async (): Promise<Account> => {
    const response = await fetch(`${config.apiHost}/me`, {
        headers: getHeaders(),
    });
    const account = await response.json();
    setAccount(account);
    return account;
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
