import type { UploadedFile } from "$lib/types";

export function bytesToMB(bytes: number) {
    return Number((bytes / 1000 / 1000).toFixed(2));
}

/**
 * Human-readable byte size using binary units. Note this uses a 1024 base while
 * bytesToMB above uses 1000 -- they are intentionally not unified, as changing either
 * would shift numbers already displayed elsewhere.
 */
export function formatBytes(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}
