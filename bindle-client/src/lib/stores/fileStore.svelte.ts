import { type UploadedFile } from '$lib/types';
import { getMe } from '$lib/services/api.svelte';
import { newAccountId, setAccount } from './accountStore.client.svelte';

let files = $state<UploadedFile[]>([]);
let selectedFile = $state<UploadedFile | null>(null);
let fileModalOpen = $state(false);

export const getFiles = () => files;
export const setFiles = (newFiles: UploadedFile[]) => {
    files = newFiles;
};

export const addFile = (file: UploadedFile) => {
    const existingFileIndex = files.findIndex(f => f.fileId === file.fileId);
    if (existingFileIndex >= 0) {
        files[existingFileIndex] = file;
    } else {
        files.push(file);
    }
};

export const deleteFile = (fileId: string) => {
    files = files.filter(f => f.fileId !== fileId);
};

export const refreshMe = async () => {
    try {
        const meResponse = await getMe();

        if (meResponse.user.files) {
            setFiles(meResponse.user.files);
        } else {
            setFiles([]);
        }
    } catch (error) {
        console.error('Error fetching files:', error);
        setFiles([]);
    }
};

export const getFileModalOpen = () => fileModalOpen;
export const setFileModalOpen = (open: boolean) => {
    fileModalOpen = open;
};

export const getSelectedFile = () => selectedFile;
export const setSelectedFile = (file: UploadedFile | null) => {
    selectedFile = file;
};
