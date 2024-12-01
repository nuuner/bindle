import type { UploadedFile } from "$lib/types";

export function bytesToMB(bytes: number) {
    return Number((bytes / 1000 / 1000).toFixed(2));
}
