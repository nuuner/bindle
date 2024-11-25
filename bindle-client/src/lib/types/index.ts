export interface UploadedFile {
    id: string;
    fileName: string;
    /**
     * Size in bytes
     */
    size: number;
    type: FileType;
    mimeType: string;
    url: string;
    details?: string;
}

export enum FileType {
    text = "text",
    image = "image",
    video = "video",
    audio = "audio",
    unknown = "unknown",
}

export interface User {
    id: string;
    files: UploadedFile[];
}
