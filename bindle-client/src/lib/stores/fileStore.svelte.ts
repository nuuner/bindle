import { type UploadedFile } from '$lib/types';
import { getMe } from '$lib/services/api.svelte';

let files = $state<UploadedFile[]>([]);
let selectedFile = $state<UploadedFile | null>(null);
let fileModalOpen = $state(false);

export const getFiles = () => files;
export const setFiles = (newFiles: UploadedFile[]) => {
    files = newFiles;
};
export const addFile = (file: UploadedFile) => {
    const existingFileIndex = files.findIndex(f => f.id === file.id);
    if (existingFileIndex >= 0) {
        files[existingFileIndex] = file;
    } else {
        files.push(file);
    }
};

export const fetchFiles = async () => {
    try {
        const meResponse = await getMe();
        if (meResponse.files) {
            setFiles(meResponse.files);
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
