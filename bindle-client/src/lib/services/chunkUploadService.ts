import { updateUploadingFile } from '../stores/uploadStore.svelte';
import { getHeaders } from './fileService';
import { withCredentials } from './accountService';

// Only used to size the initial request. The server pins the authoritative chunk
// layout at init and returns it; the upload loop below uses that, since the server
// rejects any chunk that does not fit the slice it expects.
const DEFAULT_CHUNK_SIZE = 10 * 1024 * 1024; // 10 MB

// A chunk is streamed straight through to object storage, so the server cannot replay it
// on a transient storage error the way it could when it held the whole chunk in memory -
// the retry has to come from here. Measured against the real bucket during a bad spell,
// single chunks needed three attempts, so the budget is set above what was observed
// rather than at it.
const MAX_RETRIES = 5;

// How many chunks are in flight at once. Uploading them one after another left the
// connection idle for a full round trip plus the server's work on every chunk, which on
// a link with any real latency cost more than the transfer itself.
//
// Raising this does not buy proportional bandwidth: over HTTP/2, which is what the
// deployment serves, all of these are streams on a single TCP connection and share its
// congestion window. What the concurrency actually removes is the stall between chunks,
// and four is enough for that. Going higher only helps where each request gets its own
// connection, which the browser is not doing.
const CONCURRENT_CHUNKS = 4;

export interface ChunkUploadSession {
	sessionId: string;
	chunkSize: number;
	totalChunks: number;
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
		// Initialize upload session
		const session = await initChunkedUpload(file, Math.ceil(file.size / DEFAULT_CHUNK_SIZE));
		if (!session) {
			return { success: false, error: 'Failed to initialize upload session' };
		}

		// The server decides how the file is split; follow its layout, not ours.
		const chunkSize = session.chunkSize || DEFAULT_CHUNK_SIZE;
		const totalChunks = session.totalChunks || Math.ceil(file.size / chunkSize);

		// Update upload store with session info
		updateUploadingFile(uploadId, {
			sessionId: session.sessionId,
			totalChunks,
			currentChunk: 0
		});

		const failure = await uploadAllChunks(file, uploadId, session.sessionId, chunkSize, totalChunks);
		if (failure) {
			await abortChunkedUpload(session.sessionId);
			return { success: false, error: failure };
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
 * Upload every chunk, keeping CONCURRENT_CHUNKS of them in flight. Returns an error
 * message if the upload could not be completed, or null on success.
 */
async function uploadAllChunks(
	file: File,
	uploadId: string,
	sessionId: string,
	chunkSize: number,
	totalChunks: number
): Promise<string | null> {
	const startTime = Date.now();
	let uploadedBytes = 0;
	let completedChunks = 0;
	let nextChunk = 0;
	let failure: string | null = null;

	// Chunks that landed inside the speed window, as [completedAt, bytes]. The reported
	// speed used to be the average over the whole upload, which meant a slow patch early
	// on - a congested link, a storage hiccup - kept the number low for minutes after the
	// upload had recovered, and read as "still slow" when it no longer was.
	const recent: Array<[number, number]> = [];

	// The first failure cancels the chunks still in flight rather than letting them run
	// out against a session that is about to be aborted.
	const controller = new AbortController();

	const worker = async () => {
		while (!controller.signal.aborted) {
			const chunkNumber = nextChunk++;
			if (chunkNumber >= totalChunks) {
				return;
			}

			const start = chunkNumber * chunkSize;
			const end = Math.min(start + chunkSize, file.size);
			const chunk = file.slice(start, end);

			const success = await uploadChunkWithRetry(
				sessionId,
				chunkNumber,
				chunk,
				MAX_RETRIES,
				controller.signal
			);

			if (!success) {
				if (!controller.signal.aborted) {
					failure = `Failed to upload chunk ${chunkNumber + 1}/${totalChunks}`;
					controller.abort();
				}
				return;
			}

			// Chunks finish out of order, so progress is the total uploaded so far
			// rather than anything derived from the chunk index.
			uploadedBytes += chunk.size;
			completedChunks++;

			updateUploadingFile(uploadId, {
				uploadedBytes,
				progress: Math.round((uploadedBytes / file.size) * 100),
				speed: recordSpeedSample(recent, chunk.size, startTime),
				currentChunk: completedChunks
			});
		}
	};

	await Promise.all(
		Array.from({ length: Math.min(CONCURRENT_CHUNKS, totalChunks) }, () => worker())
	);

	return failure;
}

// How far back the reported speed looks. Long enough that finishing one 10 MB chunk
// does not swing the number around, short enough that it reflects the link as it is now.
const SPEED_WINDOW_MS = 15000;

/**
 * Record a completed chunk and return the upload speed over the recent window, in bytes
 * per second. Falls back to the average since the start until the window has filled,
 * which is the best estimate available that early.
 */
function recordSpeedSample(
	recent: Array<[number, number]>,
	bytes: number,
	startTime: number
): number {
	const now = Date.now();
	recent.push([now, bytes]);

	const cutoff = now - SPEED_WINDOW_MS;
	while (recent.length > 1 && recent[0][0] < cutoff) {
		recent.shift();
	}

	// Measure from the sample before the window, since the bytes of the first sample in
	// it were transferred over the interval leading up to it, not after it.
	const windowStart = Math.max(startTime, cutoff);
	const seconds = (now - windowStart) / 1000;
	if (seconds <= 0) {
		return 0;
	}

	const windowBytes = recent.reduce((total, [, size]) => total + size, 0);
	return windowBytes / seconds;
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
			...withCredentials,
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
	retriesLeft: number,
	signal: AbortSignal
): Promise<boolean> {
	try {
		const response = await fetch(
			`/api/file/chunk/${sessionId}/${chunkNumber}`,
			{
				...withCredentials,
				method: 'POST',
				headers: getHeaders(false),
				body: chunk,
				signal
			}
		);

		if (!response.ok) {
			throw new Error(`Chunk upload failed: ${response.statusText}`);
		}

		return true;
	} catch (error) {
		// Another chunk already failed and took the session down with it; retrying this
		// one would only add requests against a session that is being aborted.
		if (signal.aborted) {
			return false;
		}

		console.error(
			`Failed to upload chunk ${chunkNumber}, retries left: ${retriesLeft}`,
			error
		);

		if (retriesLeft > 0) {
			// Wait before retrying (exponential backoff)
			await new Promise((resolve) =>
				setTimeout(resolve, (MAX_RETRIES - retriesLeft + 1) * 1000)
			);
			return uploadChunkWithRetry(sessionId, chunkNumber, chunk, retriesLeft - 1, signal);
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
			...withCredentials,
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
			...withCredentials,
			method: 'DELETE',
			headers: getHeaders(true)
		});
	} catch (error) {
		console.error('Failed to abort chunked upload:', error);
	}
}
