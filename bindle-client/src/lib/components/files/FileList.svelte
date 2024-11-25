<script lang="ts">
    import { Row, Column } from "carbon-components-svelte";
    import FileListItem from "./FileListItem.svelte";
    import {
        getFiles,
        setFileModalOpen,
        setSelectedFile,
    } from "$lib/stores/fileStore.svelte";
    import type { UploadedFile } from "$lib/types";

    let onFileClick = (file: UploadedFile) => {
        setSelectedFile(file);
        setFileModalOpen(true);
    };
</script>

{#each getFiles() as file (file.id)}
    <FileListItem {file} onClick={onFileClick} />
{/each}
