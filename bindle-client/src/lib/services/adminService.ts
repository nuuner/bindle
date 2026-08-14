import { config } from "$lib/config";

export interface AdminUser {
    accountId: string;
    fileCount: number;
    storageUsage: number;
    lastLogin: string;
    ipAddresses: string[];
}

export interface AdminFile {
    fileId: string;
    fileName: string;
    filePath: string;
    size: number;
    type: string;
    mimeType: string;
    ownerId: number;
    accountId: string;
    chunkCount: number;
    createdAt: string;
}

/**
 * System-wide overview totals. Uploads are content-addressed, so fileRecords counts
 * database rows while uniqueFiles counts the blobs actually stored -- likewise
 * logicalBytes vs storedBytes.
 */
export interface AdminStats {
    fileRecords: number;
    uniqueFiles: number;
    logicalBytes: number;
    storedBytes: number;
    dedupSavedBytes: number;
    totalUsers: number;
    usersWithFiles: number;
    averageFileBytes: number;
    largestFileBytes: number;
    storageBackend: string;
}

const getAdminHeaders = (password: string) => {
    return {
        'Content-Type': 'application/json',
        'X-Admin-Password': password,
    };
};

export const adminService = {
    async getStats(password: string): Promise<AdminStats> {
        const response = await fetch(`${config.apiHost}/admin/stats`, {
            headers: getAdminHeaders(password),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch stats');
        }

        return response.json();
    },

    async getAllUsers(password: string): Promise<AdminUser[]> {
        const response = await fetch(`${config.apiHost}/admin/users`, {
            headers: getAdminHeaders(password),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch users');
        }

        return response.json();
    },

    async getAllFiles(password: string): Promise<AdminFile[]> {
        const response = await fetch(`${config.apiHost}/admin/files`, {
            headers: getAdminHeaders(password),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch files');
        }

        return response.json();
    },

    async deleteFile(password: string, fileId: string): Promise<void> {
        const response = await fetch(`${config.apiHost}/admin/files/${fileId}`, {
            method: 'DELETE',
            headers: getAdminHeaders(password),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete file');
        }
    },

    async deleteUserFiles(password: string, accountId: string): Promise<{ count: number }> {
        const response = await fetch(`${config.apiHost}/admin/users/${accountId}/files`, {
            method: 'DELETE',
            headers: getAdminHeaders(password),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete user files');
        }

        return response.json();
    },

    async deleteAllFiles(password: string): Promise<{ recordsDeleted: number; physicalDeleted: number; physicalFailed: number }> {
        const response = await fetch(`${config.apiHost}/admin/files`, {
            method: 'DELETE',
            headers: getAdminHeaders(password),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete all files');
        }

        return response.json();
    },

    async verifyPassword(password: string): Promise<boolean> {
        try {
            await this.getAllUsers(password);
            return true;
        } catch {
            return false;
        }
    },
};
