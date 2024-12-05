export interface UploadedFile {
    fileId: string;
    fileName: string;
    /**
     * Size in bytes
     */
    size: number;
    type: FileType;
    mimeType: string;
    url: string;
    details?: string;
    createdAt: Date;
}

export enum FileType {
    text = "text",
    image = "image",
    video = "video",
    audio = "audio",
    unknown = "unknown",
}

export interface User {
    accountId: string;
    files: UploadedFile[];
}

export interface Account {
    user: User;
    uploadedBytes: number;
    uploadLimitBytes: number;
    maxFileSizeBytes: number;
}
