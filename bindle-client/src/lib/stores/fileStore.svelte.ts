import { type UploadedFile } from '$lib/types';

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

export const getFileModalOpen = () => fileModalOpen;
export const setFileModalOpen = (open: boolean) => {
    fileModalOpen = open;
};

export const getSelectedFile = () => selectedFile;
export const setSelectedFile = (file: UploadedFile | null) => {
    selectedFile = file;
};
