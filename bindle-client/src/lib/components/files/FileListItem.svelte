<script lang="ts">
    import { Button, Tooltip, Truncate } from "carbon-components-svelte";
    import {
        Document,
        Image,
        Video,
        Music,
        Copy,
        TrashCan,
        DocumentBlank,
        Checkmark,
        Close,
    } from "carbon-icons-svelte";
    import { FileType } from "$lib/types";
    import { bytesToMB } from "$lib/utils/fileUtils";
    import { copyToClipboard } from "$lib/utils/clipboard";

    let { file, onClick, deleteFile } = $props();

    let mousePosition = $state({ x: 0, y: 0 });

    let open = $state(false);

    let fileDeleteConfirmation = $state(false);
</script>

<div
    class="grid gap-4 grid-cols-[30px_minmax(0,1fr)_80px_100px] hover:bg-zinc-900 w-full"
>
    <div class="flex items-center justify-center">
        {#if file.type === FileType.text}
            <Document size={20} />
        {:else if file.type === FileType.image}
            <Image size={20} />
        {:else if file.type === FileType.video}
            <Video size={20} />
        {:else if file.type === FileType.audio}
            <Music size={20} />
        {:else}
            <DocumentBlank size={20} />
        {/if}
    </div>
    <button
        type="button"
        class="flex items-center text-left bg-transparent border-0 cursor-pointer"
        onclick={() => onClick(file)}
        onkeydown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
                onClick(file);
            }
        }}
        onmouseover={(e) => {
            open = true;
        }}
        onmousemove={(e) => {
            mousePosition = { x: e.clientX, y: e.clientY };
        }}
        onmouseleave={() => {
            open = false;
        }}
        onfocus={() => {
            open = true;
        }}
    >
        {#if open && (file.type === FileType.image || file.type === FileType.video)}
            <div
                class="fixed top-0 left-0 pointer-events-none p-2 bg-zinc-900 z-50"
                style:top={`${mousePosition.y + 20}px`}
                style:left={`${mousePosition.x + 20}px`}
            >
                {#if file.type === FileType.image}
                    <img
                        src={file.url}
                        width={150}
                        height={150}
                        alt="File preview"
                    />
                {:else}
                    <video
                        src={file.url}
                        width={150}
                        height={150}
                        muted
                        autoplay
                        loop
                    ></video>
                {/if}
            </div>
        {/if}
        <Truncate>
            {file.fileName}
        </Truncate>
    </button>
    <div class="text-sm text-right flex items-center">
        {bytesToMB(file.size).toFixed(2)} MB
    </div>
    <div class="flex items-center">
        <Button
            icon={Copy}
            size="small"
            kind="ghost"
            iconDescription="Copy link"
            tooltipPosition="left"
            on:click={() => {
                copyToClipboard(file.url);
                (document.activeElement as HTMLElement)?.blur();
            }}
        />
        {#if !fileDeleteConfirmation}
            <Button
                icon={TrashCan}
                size="small"
                kind="ghost"
                iconDescription="Delete file"
                tooltipPosition="right"
                on:click={() => {
                    fileDeleteConfirmation = true;
                }}
            />
        {:else}
            <Button
                icon={Checkmark}
                size="small"
                kind="ghost"
                iconDescription="Confirm delete"
                on:click={() => {
                    deleteFile(file);
                    fileDeleteConfirmation = false;
                }}
            />
            <Button
                icon={Close}
                size="small"
                kind="ghost"
                iconDescription="Cancel delete"
                on:click={() => {
                    fileDeleteConfirmation = false;
                }}
            />
        {/if}
    </div>
</div>
