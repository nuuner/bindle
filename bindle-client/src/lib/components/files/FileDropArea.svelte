<script lang="ts">
    import { fileService } from "$lib/services/api.svelte";

    let isDragging = false;
    let dragCounter = 0;

    function handleDragEnter(e: DragEvent) {
        e.preventDefault();
        dragCounter++;
        if (e.dataTransfer?.items && e.dataTransfer.items.length > 0) {
            isDragging = true;
        }
    }

    function handleDragLeave(e: DragEvent) {
        e.preventDefault();
        dragCounter--;
        if (dragCounter === 0) {
            isDragging = false;
        }
    }

    function handleDragOver(e: DragEvent) {
        e.preventDefault();
    }

    function handleDrop(e: DragEvent) {
        e.preventDefault();
        isDragging = false;
        dragCounter = 0;

        const files = e.dataTransfer?.files;
        if (files && files.length > 0) {
            const file = files[0];
            fileService.uploadFile(file);
        }
    }
</script>

<svelte:window
    on:dragenter={handleDragEnter}
    on:dragleave={handleDragLeave}
    on:dragover={handleDragOver}
    on:drop={handleDrop}
/>

{#if isDragging}
    <div
        class="fixed inset-0 bg-zinc-900/50 backdrop-blur-sm flex items-center justify-center z-50"
    >
        <div
            class="bg-white rounded-lg shadow-lg p-8 text-xl font-semibold text-gray-800"
        >
            Drop file to upload it
        </div>
    </div>
{/if}
