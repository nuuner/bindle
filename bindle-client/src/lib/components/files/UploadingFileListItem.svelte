<script lang="ts">
    import { InlineLoading, ProgressBar } from "carbon-components-svelte";
    import DocumentBlank from "carbon-icons-svelte/lib/DocumentBlank.svelte";
    // Imported as ErrorIcon: the icon is exported as `Error`, which would shadow the
    // global Error constructor for the rest of this module.
    import ErrorIcon from "carbon-icons-svelte/lib/Error.svelte";
    import Checkmark from "carbon-icons-svelte/lib/Checkmark.svelte";
    import { bytesToMB } from "$lib/utils/fileUtils";
    import type { UploadingFile } from "$lib/stores/uploadStore.svelte";

    let { file } = $props<{ file: UploadingFile }>();

    function formatSpeed(bytesPerSecond: number): string {
        const mbps = bytesPerSecond / (1024 * 1024);
        return mbps > 0 ? `${mbps.toFixed(2)} MB/s` : '—';
    }

    function formatChunkProgress(): string {
        if (file.currentChunk !== undefined && file.totalChunks !== undefined) {
            return `Chunk ${file.currentChunk}/${file.totalChunks}`;
        }
        return '';
    }
</script>

<div class="flex flex-col gap-2 w-full" class:opacity-70={file.status === 'uploading'}>
    <div class="grid gap-4 grid-cols-[30px_minmax(0,1fr)_110px_auto] w-full">
        <div class="flex items-center justify-center">
            {#if file.status === 'error'}
                <ErrorIcon size={20} class="text-carbon-error" />
            {:else if file.status === 'completed'}
                <Checkmark size={20} class="text-carbon-success" />
            {:else}
                <DocumentBlank size={20} />
            {/if}
        </div>
        <div class="flex items-center truncate">
            {file.fileName}
        </div>
        <div class="flex items-center">
            {bytesToMB(file.size).toFixed(2)} MB
        </div>
        <div class="flex items-center gap-2">
            {#if file.status === 'uploading'}
                <InlineLoading description={formatChunkProgress() || 'Uploading...'} />
            {:else if file.status === 'error'}
                <span class="text-carbon-error text-sm">{file.error || 'Error'}</span>
            {:else if file.status === 'completed'}
                <span class="text-carbon-success text-sm">Complete</span>
            {/if}
        </div>
    </div>

    {#if file.status === 'uploading' && file.progress > 0}
        <div class="flex flex-col gap-1 ml-[46px]">
            <ProgressBar value={file.progress} max={100} size="sm" />
            <div class="flex justify-between text-xs text-carbon-text-helper">
                <span>{file.progress}%</span>
                <span>{formatSpeed(file.speed)}</span>
            </div>
        </div>
    {/if}
</div>
