import { config } from "$lib/config";
import { getAccountId } from "$lib/stores/accountStore.client.svelte";
import { addFile, deleteFile } from "$lib/stores/fileStore.svelte";
import type { User } from '$lib/types';

const getHeaders = () => ({
    Authorization: getAccountId() ?? "",
});

export const getFilesFromServer = async () => {
    const response = await fetch(`${config.apiHost}/files`, {
        headers: getHeaders(),
    });
    return response.json();
};

export const getMe = async (): Promise<User> => {
    const response = await fetch(`${config.apiHost}/me`, {
        headers: getHeaders(),
    });
    return response.json();
};

export const uploadFile = async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${config.apiHost}/file`, {
        method: "POST",
        headers: getHeaders(),
        body: formData,
    });
    const uploadedFile = await response.json();
    addFile(uploadedFile);
    return uploadedFile;
};

export const eraseFile = async (fileId: string) => {
    const response = await fetch(`${config.apiHost}/file/${fileId}`, {
        method: "DELETE",
        headers: getHeaders(),
    });
    deleteFile(fileId);
    return response.json();
};
