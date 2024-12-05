<script lang="ts">
    import FileListItem from "./FileListItem.svelte";
    import UploadingFileListItem from "./UploadingFileListItem.svelte";
    import {
        getFiles,
        setFileModalOpen,
        setSelectedFile,
    } from "$lib/stores/fileStore.svelte";
    import { getUploadingFiles } from "$lib/stores/uploadStore.svelte";
    import type { UploadedFile } from "$lib/types";
    import { fileService } from "$lib/services/api.svelte";

    let onFileClick = (file: UploadedFile) => {
        setSelectedFile(file);
        setFileModalOpen(true);
    };

    let deleteFile = (file: UploadedFile) => {
        console.log("deleting file", file.fileId);
        fileService.deleteFile(file.fileId);
    };

    let files = $derived(
        [...getFiles()].sort(
            (a, b) =>
                new Date(b.createdAt).getTime() -
                new Date(a.createdAt).getTime(),
        ),
    );
</script>

<div class="w-full pt-2">
    {#each getUploadingFiles() as file (file.id)}
        <UploadingFileListItem {file} />
    {/each}
    {#each files as file (file.fileId)}
        <FileListItem {file} onClick={onFileClick} {deleteFile} />
    {/each}
</div>
