<script lang="ts">
    import {
        Button,
        Tooltip,
        TooltipDefinition,
    } from "carbon-components-svelte";
    import {
        Document,
        Image,
        Video,
        Music,
        Download,
        Copy,
        TrashCan,
    } from "carbon-icons-svelte";
    import type { UploadedFile } from "$lib/types";
    import { FileType } from "$lib/types";
    import { bytesToMB } from "$lib/utils/fileUtils";

    export let file: UploadedFile;
    export let onClick: (file: UploadedFile) => void;

    let open = false;
</script>

<div class="grid gap-4 grid-cols-[30px_auto_10%_110px] hover:bg-zinc-900">
    <div class="flex items-center pl-2">
        {#if file.type === FileType.text}
            <Document size={20} />
        {:else if file.type === FileType.image}
            <Image size={20} />
        {:else if file.type === FileType.video}
            <Video size={20} />
        {:else if file.type === FileType.audio}
            <Music size={20} />
        {/if}
    </div>
    <button
        type="button"
        class="flex items-center text-left w-full bg-transparent border-0 cursor-pointer"
        on:click={() => onClick(file)}
        on:keydown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
                onClick(file);
            }
        }}
        on:mouseover={() => {
            open = true;
        }}
        on:mouseleave={() => {
            open = false;
        }}
        on:focus={() => {
            open = true;
        }}
    >
        {#if file.type === FileType.image}
            <Tooltip
                hideIcon
                {open}
                direction="left"
                align="center"
                class="flex justify-center"
            >
                {#if open}
                    <img
                        src={file.url}
                        width={100}
                        height={100}
                        alt="File preview"
                    />
                {/if}
            </Tooltip>
        {/if}
        {file.fileName}
    </button>
    <div class="text-sm text-right flex items-center">
        {bytesToMB(file.size).toFixed(2)} MB
    </div>
    <div class="flex items-center">
        <Button
            icon={Download}
            size="small"
            kind="ghost"
            iconDescription="Download file"
            tooltipPosition="right"
        />
        <Button
            icon={Copy}
            size="small"
            kind="ghost"
            iconDescription="Copy link"
            tooltipPosition="right"
        />
        <Button
            icon={TrashCan}
            size="small"
            kind="ghost"
            iconDescription="Delete file"
            tooltipPosition="right"
        />
    </div>
</div>
