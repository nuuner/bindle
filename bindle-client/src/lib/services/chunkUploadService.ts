import { updateUploadingFile } from '../stores/uploadStore.svelte';
import { getHeaders } from './fileService';

const CHUNK_SIZE = 10 * 1024 * 1024; // 10 MB
const MAX_RETRIES = 3;

export interface ChunkUploadSession {
	sessionId: string;
	chunkSize: number;
}

export interface ChunkUploadResult {
	success: boolean;
	file?: any;
	error?: string;
}

/**
 * Upload a file using chunked upload
 */
export async function uploadFileChunked(
	file: File,
	uploadId: string
): Promise<ChunkUploadResult> {
	try {
		// Calculate total chunks
		const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

		// Initialize upload session
		const session = await initChunkedUpload(file, totalChunks);
		if (!session) {
			return { success: false, error: 'Failed to initialize upload session' };
		}

		// Update upload store with session info
		updateUploadingFile(uploadId, {
			sessionId: session.sessionId,
			totalChunks,
			currentChunk: 0
		});

		// Upload chunks sequentially
		const startTime = Date.now();
		let uploadedBytes = 0;

		for (let chunkNumber = 0; chunkNumber < totalChunks; chunkNumber++) {
			const start = chunkNumber * CHUNK_SIZE;
			const end = Math.min(start + CHUNK_SIZE, file.size);
			const chunk = file.slice(start, end);

			// Upload chunk with retry logic
			const success = await uploadChunkWithRetry(
				session.sessionId,
				chunkNumber,
				chunk,
				MAX_RETRIES
			);

			if (!success) {
				// Abort upload on failure
				await abortChunkedUpload(session.sessionId);
				return {
					success: false,
					error: `Failed to upload chunk ${chunkNumber + 1}/${totalChunks}`
				};
			}

			// Update progress
			uploadedBytes += chunk.size;
			const progress = Math.round((uploadedBytes / file.size) * 100);
			const elapsedSeconds = (Date.now() - startTime) / 1000;
			const speed = elapsedSeconds > 0 ? uploadedBytes / elapsedSeconds : 0;

			updateUploadingFile(uploadId, {
				uploadedBytes,
				progress,
				speed,
				currentChunk: chunkNumber + 1
			});
		}

		// Complete the upload
		const result = await completeChunkedUpload(session.sessionId);
		if (!result) {
			return { success: false, error: 'Failed to complete upload' };
		}

		updateUploadingFile(uploadId, {
			progress: 100,
			status: 'completed'
		});

		return { success: true, file: result };
	} catch (error: any) {
		console.error('Chunk upload error:', error);
		return { success: false, error: error.message || 'Upload failed' };
	}
}

/**
 * Initialize a chunked upload session
 */
async function initChunkedUpload(
	file: File,
	totalChunks: number
): Promise<ChunkUploadSession | null> {
	try {
		const response = await fetch('/api/file/chunk/init', {
			method: 'POST',
			headers: getHeaders(true),
			body: JSON.stringify({
				fileName: file.name,
				fileSize: file.size,
				mimeType: file.type,
				totalChunks
			})
		});

		if (!response.ok) {
			throw new Error(`Init failed: ${response.statusText}`);
		}

		return await response.json();
	} catch (error) {
		console.error('Failed to initialize chunked upload:', error);
		return null;
	}
}

/**
 * Upload a single chunk with retry logic
 */
async function uploadChunkWithRetry(
	sessionId: string,
	chunkNumber: number,
	chunk: Blob,
	retriesLeft: number
): Promise<boolean> {
	try {
		const response = await fetch(
			`/api/file/chunk/${sessionId}/${chunkNumber}`,
			{
				method: 'POST',
				headers: getHeaders(false),
				body: chunk
			}
		);

		if (!response.ok) {
			throw new Error(`Chunk upload failed: ${response.statusText}`);
		}

		return true;
	} catch (error) {
		console.error(
			`Failed to upload chunk ${chunkNumber}, retries left: ${retriesLeft}`,
			error
		);

		if (retriesLeft > 0) {
			// Wait before retrying (exponential backoff)
			await new Promise((resolve) =>
				setTimeout(resolve, (MAX_RETRIES - retriesLeft + 1) * 1000)
			);
			return uploadChunkWithRetry(sessionId, chunkNumber, chunk, retriesLeft - 1);
		}

		return false;
	}
}

/**
 * Complete the chunked upload
 */
async function completeChunkedUpload(sessionId: string): Promise<any | null> {
	try {
		const response = await fetch(`/api/file/chunk/${sessionId}/complete`, {
			method: 'POST',
			headers: getHeaders(true)
		});

		if (!response.ok) {
			throw new Error(`Complete failed: ${response.statusText}`);
		}

		return await response.json();
	} catch (error) {
		console.error('Failed to complete chunked upload:', error);
		return null;
	}
}

/**
 * Abort a chunked upload session
 */
export async function abortChunkedUpload(sessionId: string): Promise<void> {
	try {
		await fetch(`/api/file/chunk/${sessionId}`, {
			method: 'DELETE',
			headers: getHeaders(true)
		});
	} catch (error) {
		console.error('Failed to abort chunked upload:', error);
	}
}
