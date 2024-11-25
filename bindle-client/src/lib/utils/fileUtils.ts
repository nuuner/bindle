import type { UploadedFile } from "$lib/types";

export function bytesToMB(bytes: number) {
    return Number((bytes / 1000 / 1000).toFixed(2));
}

export function downloadFile(file: UploadedFile) {
    const link = document.createElement('a');
    link.href = file.url;
    
    link.setAttribute('download', file.fileName);
    
    link.style.display = 'none';
    
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}
