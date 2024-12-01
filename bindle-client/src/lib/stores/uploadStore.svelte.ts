export interface UploadingFile {
    id: string;
    fileName: string;
    size: number;
}

let uploadingFiles = $state<UploadingFile[]>([]);

export const getUploadingFiles = () => uploadingFiles;
export const addUploadingFile = (file: File) => {
    const uploadingFile = {
        id: crypto.randomUUID(),
        fileName: file.name,
        size: file.size,
    };
    uploadingFiles = [uploadingFile, ...uploadingFiles];
    return uploadingFile.id;
};
export const removeUploadingFile = (id: string) => {
    uploadingFiles = uploadingFiles.filter(f => f.id !== id);
}; 