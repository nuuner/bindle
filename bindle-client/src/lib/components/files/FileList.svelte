<script lang="ts">
    import FileListItem from "./FileListItem.svelte";
    import {
        getFiles,
        setFileModalOpen,
        setSelectedFile,
    } from "$lib/stores/fileStore.svelte";
    import type { UploadedFile } from "$lib/types";
    import { eraseFile } from "$lib/services/api.svelte";

    let onFileClick = (file: UploadedFile) => {
        setSelectedFile(file);
        setFileModalOpen(true);
    };

    let deleteFile = (fileId: string) => {
        eraseFile(fileId);
    };
</script>

{#each getFiles() as file (file.id)}
    <FileListItem {file} onClick={onFileClick} deleteFile={deleteFile} />
{/each}
