export type UploadStatus = 'uploading' | 'paused' | 'error' | 'completed';

export interface UploadingFile {
    id: string;
    fileName: string;
    size: number;
    uploadedBytes: number;
    progress: number; // 0-100
    speed: number; // bytes per second
    status: UploadStatus;
    sessionId?: string;
    currentChunk?: number;
    totalChunks?: number;
    error?: string;
}

let uploadingFiles = $state<UploadingFile[]>([]);

export const getUploadingFiles = () => uploadingFiles;

export const addUploadingFile = (file: File) => {
    const uploadingFile: UploadingFile = {
        id: crypto.randomUUID(),
        fileName: file.name,
        size: file.size,
        uploadedBytes: 0,
        progress: 0,
        speed: 0,
        status: 'uploading',
    };
    uploadingFiles = [uploadingFile, ...uploadingFiles];
    return uploadingFile.id;
};

export const updateUploadingFile = (id: string, updates: Partial<UploadingFile>) => {
    uploadingFiles = uploadingFiles.map(f =>
        f.id === id ? { ...f, ...updates } : f
    );
};

export const removeUploadingFile = (id: string) => {
    uploadingFiles = uploadingFiles.filter(f => f.id !== id);
}; 